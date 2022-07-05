package cluster_information

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"reflect"
	"sort"
	"sync"

	"github.com/ferry-proxy/api/apis/ferry/v1alpha1"
	versioned "github.com/ferry-proxy/client-go/generated/clientset/versioned"
	externalversions "github.com/ferry-proxy/client-go/generated/informers/externalversions"
	"github.com/ferry-proxy/ferry/pkg/client"
	"github.com/ferry-proxy/utils/objref"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

type ClusterInformationControllerConfig struct {
	Logger    logr.Logger
	Config    *restclient.Config
	Namespace string
	SyncFunc  func()
}
type ClusterInformationController struct {
	mut                     sync.RWMutex
	ctx                     context.Context
	logger                  logr.Logger
	config                  *restclient.Config
	clientset               *versioned.Clientset
	kubeClientset           *kubernetes.Clientset
	cacheClusterInformation map[string]*v1alpha1.ClusterInformation
	cacheClientset          map[string]*kubernetes.Clientset
	cacheService            map[string]*clusterServiceCache
	cacheTunnelPorts        map[string]*tunnelPorts
	cacheIdentity           map[string]string
	syncFunc                func()
	namespace               string
}

func NewClusterInformationController(conf ClusterInformationControllerConfig) *ClusterInformationController {
	return &ClusterInformationController{
		config:                  conf.Config,
		namespace:               conf.Namespace,
		logger:                  conf.Logger,
		syncFunc:                conf.SyncFunc,
		cacheClusterInformation: map[string]*v1alpha1.ClusterInformation{},
		cacheClientset:          map[string]*kubernetes.Clientset{},
		cacheService:            map[string]*clusterServiceCache{},
		cacheTunnelPorts:        map[string]*tunnelPorts{},
		cacheIdentity:           map[string]string{},
	}
}

func (c *ClusterInformationController) Run(ctx context.Context) error {
	c.logger.Info("ClusterInformation controller started")
	defer c.logger.Info("ClusterInformation controller stopped")

	clientset, err := versioned.NewForConfig(c.config)
	if err != nil {
		return err
	}
	c.clientset = clientset

	kubeClientset, err := kubernetes.NewForConfig(c.config)
	if err != nil {
		return err
	}
	c.kubeClientset = kubeClientset

	c.ctx = ctx
	informerFactory := externalversions.NewSharedInformerFactoryWithOptions(clientset, 0,
		externalversions.WithNamespace(c.namespace))
	informer := informerFactory.
		Ferry().
		V1alpha1().
		ClusterInformations().
		Informer()
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.onAdd,
		UpdateFunc: c.onUpdate,
		DeleteFunc: c.onDelete,
	})

	informer.Run(ctx.Done())
	return nil
}

func (c *ClusterInformationController) updateStatus(name string, phase string) error {
	ci := c.cacheClusterInformation[name]
	if ci == nil {
		return fmt.Errorf("not found ClusterInformation %s", name)
	}

	ci = ci.DeepCopy()
	ci.Status.LastSynchronizationTimestamp = metav1.Now()
	ci.Status.Phase = phase

	_, err := c.clientset.
		FerryV1alpha1().
		ClusterInformations(c.namespace).
		UpdateStatus(c.ctx, ci, metav1.UpdateOptions{})
	return err
}

func (c *ClusterInformationController) Clientset(name string) *kubernetes.Clientset {
	c.mut.RLock()
	defer c.mut.RUnlock()
	return c.cacheClientset[name]
}

func (c *ClusterInformationController) ListServices(name string) []*corev1.Service {
	c.mut.RLock()
	defer c.mut.RUnlock()
	cache := c.cacheService[name]
	if cache == nil {
		return nil
	}

	svcs := []*corev1.Service{}
	cache.ForEach(func(svc *corev1.Service) {
		svcs = append(svcs, svc)
	})

	sort.Slice(svcs, func(i, j int) bool {
		return svcs[i].CreationTimestamp.Before(&svcs[j].CreationTimestamp)
	})

	return svcs
}

func (c *ClusterInformationController) GetIdentity(name string) string {
	c.mut.Lock()
	defer c.mut.Unlock()
	return c.cacheIdentity[name]
}

func (c *ClusterInformationController) RegistryServiceCallback(exportClusterName, importClusterName string, cb func()) {
	c.mut.Lock()
	defer c.mut.Unlock()
	c.cacheService[exportClusterName].RegistryCallback(importClusterName, cb)
}

func (c *ClusterInformationController) UnregistryServiceCallback(exportClusterName, importClusterName string) {
	c.mut.Lock()
	defer c.mut.Unlock()
	c.cacheService[exportClusterName].UnregistryCallback(importClusterName)
}

func (c *ClusterInformationController) LoadPortPeer(importClusterName string, list *corev1.ServiceList) {
	c.mut.RLock()
	defer c.mut.RUnlock()
	c.cacheTunnelPorts[importClusterName].LoadPortPeer(list)
}

func (c *ClusterInformationController) GetPortPeer(importClusterName string, cluster, namespace, name string, port int32) int32 {
	c.mut.RLock()
	defer c.mut.RUnlock()
	return c.cacheTunnelPorts[importClusterName].GetPort(cluster, namespace, name, port)
}

func (c *ClusterInformationController) onAdd(obj interface{}) {
	f := obj.(*v1alpha1.ClusterInformation)
	f = f.DeepCopy()
	c.logger.Info("onAdd",
		"ClusterInformation", objref.KObj(f),
	)

	c.mut.Lock()
	defer c.mut.Unlock()

	clientset, err := client.NewClientsetFromKubeconfig(f.Spec.Kubeconfig)
	if err != nil {
		c.logger.Error(err, "NewClientsetFromKubeconfig")
		err = c.updateStatus(f.Name, "Disconnected")
		if err != nil {
			c.logger.Error(err, "UpdateStatus",
				"ClusterInformation", objref.KObj(f),
			)
		}
	} else {
		c.cacheClientset[f.Name] = clientset
		err = c.updateStatus(f.Name, "Connected")
		if err != nil {
			c.logger.Error(err, "UpdateStatus",
				"ClusterInformation", objref.KObj(f),
			)
		}
	}

	c.cacheClusterInformation[f.Name] = f
	c.cacheTunnelPorts[f.Name] = newTunnelPorts(tunnelPortsConfig{
		Logger: c.logger.WithName(f.Name),
	})

	clusterService := newClusterServiceCache(clusterServiceCacheConfig{
		Clientset: clientset,
		Logger:    c.logger.WithName(f.Name),
	})
	c.cacheService[f.Name] = clusterService

	err = clusterService.Start(c.ctx)
	if err != nil {
		c.logger.Error(err, "failed start cluster service cache")
	}

	err = c.updateIdentityKey(f.Name)
	if err != nil {
		c.logger.Error(err, "UpdateIdentityKey",
			"ClusterInformation", objref.KObj(f),
		)
	}
	c.syncFunc()
}

func (c *ClusterInformationController) updateIdentityKey(name string) error {
	secret, err := c.kubeClientset.
		CoreV1().
		Secrets(c.namespace).
		Get(c.ctx, name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if secret.Data == nil {
		return fmt.Errorf("secret %q is empty", name)
	}
	identity := secret.Data["identity"]
	if len(identity) == 0 {
		return fmt.Errorf("secret %q not found identity key", name)
	}
	c.cacheIdentity[name] = base64.URLEncoding.EncodeToString(identity)
	return nil
}

func (c *ClusterInformationController) onUpdate(oldObj, newObj interface{}) {
	f := newObj.(*v1alpha1.ClusterInformation)
	f = f.DeepCopy()
	c.logger.Info("onUpdate",
		"ClusterInformation", objref.KObj(f),
	)

	c.mut.Lock()
	defer c.mut.Unlock()

	err := c.updateIdentityKey(f.Name)
	if err != nil {
		c.logger.Error(err, "UpdateIdentityKey",
			"ClusterInformation", objref.KObj(f),
		)
	}

	if reflect.DeepEqual(c.cacheClusterInformation[f.Name].Spec, f.Spec) {
		c.cacheClusterInformation[f.Name] = f
		return
	}

	if !bytes.Equal(c.cacheClusterInformation[f.Name].Spec.Kubeconfig, f.Spec.Kubeconfig) {
		clientset, err := client.NewClientsetFromKubeconfig(f.Spec.Kubeconfig)
		if err != nil {
			c.logger.Error(err, "NewClientsetFromKubeconfig")
			err = c.updateStatus(f.Name, "Disconnected")
			if err != nil {
				c.logger.Error(err, "UpdateStatus",
					"ClusterInformation", objref.KObj(f),
				)
			}
		} else {
			c.cacheClientset[f.Name] = clientset
			err := c.cacheService[f.Name].ResetClientset(clientset)
			if err != nil {
				c.logger.Error(err, "Reset clientset")
				err = c.updateStatus(f.Name, "Disconnected")
				if err != nil {
					c.logger.Error(err, "UpdateStatus",
						"ClusterInformation", objref.KObj(f),
					)
				}
			} else {
				err = c.updateStatus(f.Name, "Connected")
				if err != nil {
					c.logger.Error(err, "UpdateStatus",
						"ClusterInformation", objref.KObj(f),
					)
				}
			}
		}
	}

	c.cacheClusterInformation[f.Name] = f

	c.syncFunc()
}

func (c *ClusterInformationController) onDelete(obj interface{}) {
	f := obj.(*v1alpha1.ClusterInformation)
	c.logger.Info("onDelete",
		"ClusterInformation", objref.KObj(f),
	)

	c.mut.Lock()
	defer c.mut.Unlock()

	delete(c.cacheClientset, f.Name)
	delete(c.cacheClusterInformation, f.Name)
	delete(c.cacheTunnelPorts, f.Name)

	if c.cacheService[f.Name] != nil {
		c.cacheService[f.Name].Close()
	}
	delete(c.cacheService, f.Name)
	delete(c.cacheIdentity, f.Name)

	c.syncFunc()
}

func (c *ClusterInformationController) GetClusterInformation(name string) *v1alpha1.ClusterInformation {
	c.mut.RLock()
	defer c.mut.RUnlock()
	return c.cacheClusterInformation[name]
}
