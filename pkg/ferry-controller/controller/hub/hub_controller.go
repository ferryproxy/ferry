/*
Copyright 2022 FerryProxy Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package hub

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"reflect"
	"sort"
	"sync"

	"github.com/ferryproxy/api/apis/traffic/v1alpha2"
	versioned "github.com/ferryproxy/client-go/generated/clientset/versioned"
	externalversions "github.com/ferryproxy/client-go/generated/informers/externalversions"
	"github.com/ferryproxy/ferry/pkg/client"
	"github.com/ferryproxy/ferry/pkg/consts"
	"github.com/ferryproxy/ferry/pkg/utils/objref"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

type HubControllerConfig struct {
	Logger    logr.Logger
	Config    *restclient.Config
	Namespace string
	SyncFunc  func()
}

type HubController struct {
	mut              sync.RWMutex
	ctx              context.Context
	logger           logr.Logger
	config           *restclient.Config
	clientset        *versioned.Clientset
	kubeClientset    kubernetes.Interface
	cacheHub         map[string]*v1alpha2.Hub
	cacheClientset   map[string]kubernetes.Interface
	cacheService     map[string]*clusterServiceCache
	cacheTunnelPorts map[string]*tunnelPorts
	cacheIdentity    map[string]string
	cacheKubeconfig  map[string][]byte
	syncFunc         func()
	namespace        string
}

func NewHubController(conf HubControllerConfig) *HubController {
	return &HubController{
		config:           conf.Config,
		namespace:        conf.Namespace,
		logger:           conf.Logger,
		syncFunc:         conf.SyncFunc,
		cacheHub:         map[string]*v1alpha2.Hub{},
		cacheClientset:   map[string]kubernetes.Interface{},
		cacheService:     map[string]*clusterServiceCache{},
		cacheTunnelPorts: map[string]*tunnelPorts{},
		cacheIdentity:    map[string]string{},
		cacheKubeconfig:  map[string][]byte{},
	}
}

func (c *HubController) Run(ctx context.Context) error {
	c.logger.Info("Hub controller started")
	defer c.logger.Info("Hub controller stopped")

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
		Traffic().
		V1alpha2().
		Hubs().
		Informer()
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.onAdd,
		UpdateFunc: c.onUpdate,
		DeleteFunc: c.onDelete,
	})

	informer.Run(ctx.Done())
	return nil
}

func (c *HubController) updateStatus(name string, phase string) error {
	ci := c.cacheHub[name]
	if ci == nil {
		return fmt.Errorf("not found Hub %s", name)
	}

	ci = ci.DeepCopy()
	ci.Status.LastSynchronizationTimestamp = metav1.Now()
	ci.Status.Phase = phase

	_, err := c.clientset.
		TrafficV1alpha2().
		Hubs(c.namespace).
		UpdateStatus(c.ctx, ci, metav1.UpdateOptions{})
	return err
}

func (c *HubController) Clientset(name string) kubernetes.Interface {
	c.mut.RLock()
	defer c.mut.RUnlock()
	return c.cacheClientset[name]
}

func (c *HubController) ListServices(name string) []*corev1.Service {
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

func (c *HubController) GetIdentity(name string) string {
	c.mut.Lock()
	defer c.mut.Unlock()
	ident := c.cacheIdentity[name]
	if ident != "" {
		return ident
	}

	err := c.updateIdentity(name)
	if err != nil {
		c.logger.Error(err, "failed to update identity key")
		return ""
	}
	return c.cacheIdentity[name]
}

func (c *HubController) RegistryServiceCallback(exportHubName, importHubName string, cb func()) {
	c.mut.Lock()
	defer c.mut.Unlock()
	c.cacheService[exportHubName].RegistryCallback(importHubName, cb)
}

func (c *HubController) UnregistryServiceCallback(exportHubName, importHubName string) {
	c.mut.Lock()
	defer c.mut.Unlock()
	c.cacheService[exportHubName].UnregistryCallback(importHubName)
}

func (c *HubController) LoadPortPeer(importHubName string, list *corev1.ServiceList) {
	c.mut.RLock()
	defer c.mut.RUnlock()
	c.cacheTunnelPorts[importHubName].LoadPortPeer(list)
}

func (c *HubController) GetPortPeer(importHubName string, cluster, namespace, name string, port int32) int32 {
	c.mut.RLock()
	defer c.mut.RUnlock()
	return c.cacheTunnelPorts[importHubName].GetPort(cluster, namespace, name, port)
}

func (c *HubController) onAdd(obj interface{}) {
	f := obj.(*v1alpha2.Hub)
	f = f.DeepCopy()
	c.logger.Info("onAdd",
		"hub", objref.KObj(f),
	)

	c.mut.Lock()
	defer c.mut.Unlock()

	c.cacheHub[f.Name] = f

	err := c.updateKubeconfig(f.Name)
	if err != nil {
		c.logger.Error(err, "updateKubeconfig",
			"hub", objref.KObj(f),
		)
	}

	kubeconfig := c.cacheKubeconfig[f.Name]

	if len(kubeconfig) == 0 {
		c.logger.Info("Failed get kubeconfig ", "hub", f.Name)
		return
	}

	clientset, err := client.NewClientsetFromKubeconfig(kubeconfig)
	if err != nil {
		c.logger.Error(err, "NewClientsetFromKubeconfig")
		err = c.updateStatus(f.Name, "Disconnected")
		if err != nil {
			c.logger.Error(err, "UpdateStatus",
				"hub", objref.KObj(f),
			)
		}
	} else {
		c.cacheClientset[f.Name] = clientset
		err = c.updateStatus(f.Name, "Connected")
		if err != nil {
			c.logger.Error(err, "UpdateStatus",
				"hub", objref.KObj(f),
			)
		}
	}

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

	err = c.updateIdentity(f.Name)
	if err != nil {
		c.logger.Error(err, "UpdateIdentityKey",
			"hub", objref.KObj(f),
		)
	}
	c.syncFunc()
}

func (c *HubController) updateIdentity(name string) error {
	secret, err := c.cacheClientset[name].
		CoreV1().
		Secrets(consts.FerryTunnelNamespace).
		Get(c.ctx, consts.FerryTunnelName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if secret.Data == nil {
		return fmt.Errorf("hub %q secret %s.%s is empty", name, consts.FerryTunnelName, consts.FerryTunnelNamespace)
	}
	identity := secret.Data["identity"]
	if len(identity) == 0 {
		return fmt.Errorf("hub %q not found identity key", name)
	}
	c.cacheIdentity[name] = base64.URLEncoding.EncodeToString(identity)
	return nil
}

func (c *HubController) updateKubeconfig(name string) error {
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
	kubeconfig := secret.Data["kubeconfig"]
	if len(kubeconfig) == 0 {
		return fmt.Errorf("secret %q not found kubeconfig key", name)
	}
	c.cacheKubeconfig[name] = kubeconfig
	return nil
}

func (c *HubController) onUpdate(oldObj, newObj interface{}) {
	f := newObj.(*v1alpha2.Hub)
	f = f.DeepCopy()
	c.logger.Info("onUpdate",
		"hub", objref.KObj(f),
	)

	c.mut.Lock()
	defer c.mut.Unlock()

	err := c.updateIdentity(f.Name)
	if err != nil {
		c.logger.Error(err, "UpdateIdentityKey",
			"hub", objref.KObj(f),
		)
	}

	oldKubeconfig := c.cacheKubeconfig[f.Name]
	err = c.updateKubeconfig(f.Name)
	if err != nil {
		c.logger.Error(err, "updateKubeconfig",
			"hub", objref.KObj(f),
		)
	}
	kubeconfig := c.cacheKubeconfig[f.Name]

	if reflect.DeepEqual(c.cacheHub[f.Name].Spec, f.Spec) && reflect.DeepEqual(oldKubeconfig, kubeconfig) {
		c.cacheHub[f.Name] = f
		return
	}

	if !bytes.Equal(oldKubeconfig, kubeconfig) {
		clientset, err := client.NewClientsetFromKubeconfig(kubeconfig)
		if err != nil {
			c.logger.Error(err, "NewClientsetFromKubeconfig")
			err = c.updateStatus(f.Name, "Disconnected")
			if err != nil {
				c.logger.Error(err, "UpdateStatus",
					"hub", objref.KObj(f),
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
						"hub", objref.KObj(f),
					)
				}
			} else {
				err = c.updateStatus(f.Name, "Connected")
				if err != nil {
					c.logger.Error(err, "UpdateStatus",
						"hub", objref.KObj(f),
					)
				}
			}
		}
	}

	c.cacheHub[f.Name] = f

	c.syncFunc()
}

func (c *HubController) onDelete(obj interface{}) {
	f := obj.(*v1alpha2.Hub)
	c.logger.Info("onDelete",
		"hub", objref.KObj(f),
	)

	c.mut.Lock()
	defer c.mut.Unlock()

	delete(c.cacheClientset, f.Name)
	delete(c.cacheHub, f.Name)
	delete(c.cacheTunnelPorts, f.Name)

	if c.cacheService[f.Name] != nil {
		c.cacheService[f.Name].Close()
	}
	delete(c.cacheService, f.Name)
	delete(c.cacheIdentity, f.Name)

	c.syncFunc()
}

func (c *HubController) GetHub(name string) *v1alpha2.Hub {
	c.mut.RLock()
	defer c.mut.RUnlock()
	return c.cacheHub[name]
}

func (c *HubController) GetHubGateway(hubName string, forHub string) v1alpha2.HubSpecGateway {
	hub := c.cacheHub[hubName]
	if hub != nil {
		if hub.Spec.Override != nil {
			h, ok := hub.Spec.Override[forHub]
			if ok {
				return h
			}
		}
		return hub.Spec.Gateway
	}
	return v1alpha2.HubSpecGateway{}
}
