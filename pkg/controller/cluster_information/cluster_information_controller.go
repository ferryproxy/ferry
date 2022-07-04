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
	"k8s.io/apimachinery/pkg/labels"
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

func (c *ClusterInformationController) UpdateStatus(name string, importedFrom []string, exportedTo []string, phase string) error {
	c.mut.RLock()
	defer c.mut.RUnlock()
	return c.updateStatus(name, importedFrom, exportedTo, phase)
}

func (c *ClusterInformationController) updateStatus(name string, importedFrom []string, exportedTo []string, phase string) error {
	ci := c.cacheClusterInformation[name]
	if ci == nil {
		return fmt.Errorf("not found ClusterInformation %s", name)
	}
	sort.Strings(importedFrom)
	sort.Strings(exportedTo)

	ci = ci.DeepCopy()
	ci.Status.ImportedFrom = importedFrom
	ci.Status.ExportedTo = exportedTo
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

func (c *ClusterInformationController) ServiceCache(name string) *clusterServiceCache {
	c.mut.RLock()
	defer c.mut.RUnlock()
	return c.cacheService[name]
}

func (c *ClusterInformationController) GetIdentity(name string) string {
	c.mut.Lock()
	defer c.mut.Unlock()
	return c.cacheIdentity[name]
}

func (c *ClusterInformationController) TunnelPorts(name string) *tunnelPorts {
	c.mut.RLock()
	defer c.mut.RUnlock()
	return c.cacheTunnelPorts[name]
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
	} else {
		c.cacheClientset[f.Name] = clientset
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

	err = c.updateStatus(f.Name, []string{}, []string{}, "Pending")
	if err != nil {
		c.logger.Error(err, "UpdateStatus",
			"ClusterInformation", objref.KObj(f),
		)
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
		} else {
			c.cacheClientset[f.Name] = clientset
			err := c.cacheService[f.Name].ResetClientset(clientset)
			if err != nil {
				c.logger.Error(err, "Reset clientset")
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

func (c *ClusterInformationController) Get(name string) *v1alpha1.ClusterInformation {
	c.mut.RLock()
	defer c.mut.RUnlock()
	return c.cacheClusterInformation[name]
}

func (c *ClusterInformationController) proxy(proxy v1alpha1.ClusterInformationSpecGatewayWay) (string, error) {
	if proxy.Proxy != "" {
		return proxy.Proxy, nil
	}

	ci := c.Get(proxy.ClusterName)
	if ci == nil {
		return "", fmt.Errorf("failed get cluster information %q", proxy.ClusterName)
	}
	if ci.Spec.Gateway.Address == "" {
		return "", fmt.Errorf("failed get address of cluster information %q", proxy.ClusterName)
	}
	address := ci.Spec.Gateway.Address
	return "ssh://" + address + "?identity_data=" + c.GetIdentity(proxy.ClusterName), nil
}

func (c *ClusterInformationController) Proxies(proxies v1alpha1.ClusterInformationSpecGatewayWays) ([]string, error) {
	out := make([]string, 0, len(proxies))
	for _, proxy := range proxies {
		p, err := c.proxy(proxy)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

func (c *ClusterInformationController) PoliciesToRules(policies []*v1alpha1.FerryPolicy) []*v1alpha1.MappingRule {
	svcs := []*corev1.Service{}
	out := []*v1alpha1.MappingRule{}
	rules := GroupFerryPolicies(policies)
	controller := true
	for exportClusterName, i := range rules {
		svcs = svcs[:0]
		c.ServiceCache(exportClusterName).
			ForEach(func(svc *corev1.Service) {
				svcs = append(svcs, svc)
			})
		sort.Slice(svcs, func(i, j int) bool {
			return svcs[i].CreationTimestamp.Before(&svcs[j].CreationTimestamp)
		})

		for importClusterName, matches := range i {
			for _, match := range matches {
				for _, svc := range svcs {
					var (
						exportName      = match.Export.Name
						exportNamespace = match.Export.Namespace
						importName      = match.Import.Name
						importNamespace = match.Import.Namespace
					)

					if len(match.Export.Labels) != 0 {
						if !labels.SelectorFromSet(match.Export.Labels).Matches(labels.Set(svc.Labels)) {
							continue
						}
						if exportName == "" {
							exportName = svc.Name
						}
						if exportNamespace == "" {
							exportNamespace = svc.Namespace
						}
					} else {
						if svc.Namespace != exportNamespace {
							continue
						}

						if svc.Name != exportName {
							continue
						}
					}

					if importName == "" {
						importName = exportName
					}

					if importNamespace == "" {
						importNamespace = exportNamespace
					}

					policy := match.Policy

					ports := []uint32{}
					for _, port := range svc.Spec.Ports {
						ports = append(ports, uint32(port.Port))
					}
					out = append(out, &v1alpha1.MappingRule{
						ObjectMeta: metav1.ObjectMeta{
							Name:      fmt.Sprintf("%s-%s-%s-%s-%s-%s-%s", policy.Name, exportClusterName, exportNamespace, exportName, importClusterName, importNamespace, importName),
							Namespace: policy.Namespace,
							Labels:    policy.Labels,
							OwnerReferences: []metav1.OwnerReference{
								{
									APIVersion: v1alpha1.GroupVersion.String(),
									Kind:       "FerryPolicy",
									Name:       policy.Name,
									UID:        policy.UID,
									Controller: &controller,
								},
							},
						},
						Spec: v1alpha1.MappingRuleSpec{
							Import: v1alpha1.MappingRuleSpecPorts{
								ClusterName: importClusterName,
								Service: v1alpha1.MappingRuleSpecPortsService{
									Name:      importName,
									Namespace: importNamespace,
								},
							},
							Export: v1alpha1.MappingRuleSpecPorts{
								ClusterName: exportClusterName,
								Service: v1alpha1.MappingRuleSpecPortsService{
									Name:      exportName,
									Namespace: exportNamespace,
								},
							},
						},
					})
				}
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out
}

func GroupFerryPolicies(policies []*v1alpha1.FerryPolicy) map[string]map[string][]GroupFerryPolicy {
	mapping := map[string]map[string][]GroupFerryPolicy{}
	for _, policy := range policies {
		for _, rule := range policy.Spec.Rules {
			for _, export := range rule.Exports {
				if export.ClusterName == "" {
					continue
				}
				if _, ok := mapping[export.ClusterName]; !ok {
					mapping[export.ClusterName] = map[string][]GroupFerryPolicy{}
				}
				for _, impor := range rule.Imports {
					if impor.ClusterName == "" || impor.ClusterName == export.ClusterName {
						continue
					}
					if _, ok := mapping[export.ClusterName][impor.ClusterName]; !ok {
						mapping[export.ClusterName][impor.ClusterName] = []GroupFerryPolicy{}
					}

					matchRule := GroupFerryPolicy{
						Policy: policy,
						Export: export.Match,
						Import: impor.Match,
					}
					mapping[export.ClusterName][impor.ClusterName] = append(mapping[export.ClusterName][impor.ClusterName], matchRule)
				}
			}
		}
	}
	return mapping
}

type GroupFerryPolicy struct {
	Policy *v1alpha1.FerryPolicy
	Export v1alpha1.FerryPolicySpecRuleMatch
	Import v1alpha1.FerryPolicySpecRuleMatch
}
