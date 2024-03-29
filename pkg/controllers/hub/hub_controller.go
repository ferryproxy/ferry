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
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"

	trafficv1alpha2 "github.com/ferryproxy/api/apis/traffic/v1alpha2"
	externalversions "github.com/ferryproxy/client-go/generated/informers/externalversions"
	"github.com/ferryproxy/ferry/pkg/client"
	"github.com/ferryproxy/ferry/pkg/conditions"
	"github.com/ferryproxy/ferry/pkg/consts"
	healthclient "github.com/ferryproxy/ferry/pkg/services/health/client"
	portsclient "github.com/ferryproxy/ferry/pkg/services/ports/client"
	"github.com/ferryproxy/ferry/pkg/utils/objref"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
	mcsv1alpha1 "sigs.k8s.io/mcs-api/pkg/apis/v1alpha1"
)

type HubControllerConfig struct {
	Logger    logr.Logger
	Clientset client.Interface
	Namespace string
	SyncFunc  func()
}

type HubController struct {
	mut                sync.RWMutex
	mutStatus          sync.Mutex
	ctx                context.Context
	logger             logr.Logger
	clientset          client.Interface
	cacheHub           map[string]*trafficv1alpha2.Hub
	cacheClientset     map[string]client.Interface
	cacheService       map[string]*clusterServiceCache
	cacheServiceExport map[string]*clusterServiceExportCache
	cacheServiceImport map[string]*clusterServiceImportCache
	cacheTunnelPorts   map[string]*tunnelPorts
	cacheAuthorized    map[string]string
	cacheKubeconfig    map[string][]byte
	syncFunc           func()
	namespace          string
	conditionsManager  *conditions.ConditionsManager
}

func NewHubController(conf HubControllerConfig) *HubController {
	return &HubController{
		clientset:          conf.Clientset,
		namespace:          conf.Namespace,
		logger:             conf.Logger,
		syncFunc:           conf.SyncFunc,
		cacheHub:           map[string]*trafficv1alpha2.Hub{},
		cacheClientset:     map[string]client.Interface{},
		cacheService:       map[string]*clusterServiceCache{},
		cacheServiceExport: map[string]*clusterServiceExportCache{},
		cacheServiceImport: map[string]*clusterServiceImportCache{},
		cacheTunnelPorts:   map[string]*tunnelPorts{},
		cacheAuthorized:    map[string]string{},
		cacheKubeconfig:    map[string][]byte{},
		conditionsManager:  conditions.NewConditionsManager(),
	}
}

func (c *HubController) Run(ctx context.Context) error {
	c.logger.Info("hub controller started")
	defer c.logger.Info("hub controller stopped")

	c.ctx = ctx
	informerFactory := externalversions.NewSharedInformerFactoryWithOptions(c.clientset.Ferry(), 0,
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

func (c *HubController) GetTunnelAddressInControlPlane(hubName string) string {
	host := "ferry-tunnel.ferry-tunnel-system:8080"
	if hubName != consts.ControlPlaneName {
		host = hubName + "-" + host
	}
	return host
}

func (c *HubController) UpdateHubConditions(name string, conditions []metav1.Condition) {
	c.mutStatus.Lock()
	defer c.mutStatus.Unlock()

	var retErr error
	defer func() {
		if retErr != nil {
			c.logger.Error(retErr, "failed to update status")
		}
	}()

	status := trafficv1alpha2.HubStatus{}
	status.LastSynchronizationTimestamp = metav1.Now()

	updated := false
	dur := 10 * time.Second
	for _, condition := range conditions {
		updated = updated || c.conditionsManager.SetWithDuration(name, condition, dur)
	}
	if !updated {
		return
	}

	ready, reason := c.conditionsManager.Ready(name,
		trafficv1alpha2.ConnectedCondition,
		trafficv1alpha2.TunnelHealthCondition,
	)
	if ready {
		c.conditionsManager.Set(name, metav1.Condition{
			Type:   trafficv1alpha2.HubReady,
			Status: metav1.ConditionTrue,
			Reason: trafficv1alpha2.HubReady,
		})
		status.Phase = trafficv1alpha2.HubReady
	} else {
		c.conditionsManager.Set(name, metav1.Condition{
			Type:   trafficv1alpha2.HubReady,
			Status: metav1.ConditionFalse,
			Reason: "NotReady",
		})
		status.Phase = reason
	}

	status.Conditions = c.conditionsManager.Get(name)
	data, err := json.Marshal(map[string]interface{}{
		"status": status,
	})
	if err != nil {
		retErr = err
		return
	}
	_, err = c.clientset.
		Ferry().
		TrafficV1alpha2().
		Hubs(consts.FerryNamespace).
		Patch(c.ctx, name, types.MergePatchType, data, metav1.PatchOptions{}, "status")
	if err != nil {
		retErr = err
		return
	}
}

func (c *HubController) ResetClientset(hubName string) {
	c.mut.Lock()
	defer c.mut.Unlock()
	delete(c.cacheClientset, hubName)
}

func (c *HubController) updateClientset(hubName string) (client.Interface, error) {
	hub := c.cacheHub[hubName]
	if hub == nil {
		return nil, fmt.Errorf("not found hub %q", hubName)
	}

	clientset, updated, err := c.tryConnectAndUpdateStatus(hubName)
	if err != nil {
		c.logger.Error(err, "tryConnectAndUpdateStatus")
		c.UpdateHubConditions(hubName, []metav1.Condition{
			{
				Type:    trafficv1alpha2.ConnectedCondition,
				Status:  metav1.ConditionFalse,
				Reason:  "Disconnected",
				Message: err.Error(),
			},
		})
		return nil, err
	}
	c.UpdateHubConditions(hubName, []metav1.Condition{
		{
			Type:   trafficv1alpha2.ConnectedCondition,
			Status: metav1.ConditionTrue,
			Reason: "Connected",
		},
	})

	if updated {
		c.enableCache(hubName, clientset)
		c.checkHealth(hubName)
	}

	if IsEnabledMCS(hub) {
		c.enableMCS(hub, clientset)
	} else {
		c.disableMCS(hub)
	}

	c.syncFunc()
	return clientset, nil
}

func (c *HubController) UpdateClientset(hubName string) (client.Interface, error) {
	c.mut.Lock()
	defer c.mut.Unlock()

	return c.updateClientset(hubName)
}

func (c *HubController) Clientset(hubName string) (client.Interface, error) {
	c.mut.Lock()
	defer c.mut.Unlock()

	clientset, ok := c.cacheClientset[hubName]
	if ok && clientset != nil {
		return clientset, nil
	}
	return c.updateClientset(hubName)
}

func (c *HubController) GetService(hubName string, namespace, name string) (*corev1.Service, bool) {
	c.mut.RLock()
	defer c.mut.RUnlock()
	cache := c.cacheService[hubName]
	if cache == nil {
		return nil, false
	}
	return cache.Get(namespace, name)
}

func (c *HubController) ListServices(hubName string) []*corev1.Service {
	c.mut.RLock()
	defer c.mut.RUnlock()
	cache := c.cacheService[hubName]
	if cache == nil {
		return nil
	}
	return cache.List()
}

func (c *HubController) GetAuthorized(name string) string {
	c.mut.Lock()
	defer c.mut.Unlock()
	ident := c.cacheAuthorized[name]
	if ident != "" {
		return ident
	}

	err := c.updateAuthorized(name)
	if err != nil {
		c.logger.Error(err, "failed to update authorized key")
		return ""
	}
	return c.cacheAuthorized[name]
}

func (c *HubController) LoadPortPeer(importHubName string, cluster, namespace, name string, port, bindPort int32) error {
	c.mut.RLock()
	defer c.mut.RUnlock()
	if c.cacheTunnelPorts[importHubName] == nil {
		return fmt.Errorf("failed to get load peer on hub %q", importHubName)
	}
	return c.cacheTunnelPorts[importHubName].LoadPortBind(cluster, namespace, name, port, bindPort)
}

func (c *HubController) GetPortPeer(importHubName string, cluster, namespace, name string, port int32) (int32, error) {
	c.mut.RLock()
	defer c.mut.RUnlock()
	if c.cacheTunnelPorts[importHubName] == nil {
		return 0, fmt.Errorf("failed to get port peer on hub %q", importHubName)
	}
	return c.cacheTunnelPorts[importHubName].GetPortBind(cluster, namespace, name, port)
}

func (c *HubController) DeletePortPeer(importHubName string, cluster, namespace, name string, port int32) (int32, error) {
	c.mut.RLock()
	defer c.mut.RUnlock()
	if c.cacheTunnelPorts[importHubName] == nil {
		return 0, fmt.Errorf("failed to delete port peer on hub %q", importHubName)
	}
	return c.cacheTunnelPorts[importHubName].DeletePortBind(cluster, namespace, name, port)
}

func (c *HubController) HubReady(hubName string) bool {
	return c.conditionsManager.IsTrue(hubName, trafficv1alpha2.HubReady)
}

func (c *HubController) onAdd(obj interface{}) {
	f := obj.(*trafficv1alpha2.Hub)
	f = f.DeepCopy()
	c.logger.Info("onAdd",
		"hub", objref.KObj(f),
	)

	c.mut.Lock()
	c.cacheHub[f.Name] = f
	c.enablePorts(f.Name)
	c.mut.Unlock()

	_, err := c.Clientset(f.Name)
	if err != nil {
		c.logger.Error(err, "Clientset",
			"hub", objref.KRef(consts.FerryNamespace, f.Name),
		)
	}
}

func IsEnabledMCS(f *trafficv1alpha2.Hub) bool {
	return f != nil && f.Labels != nil && f.Labels[consts.LabelMCSMarkHubKey] == consts.LabelMCSMarkHubValue
}

func (c *HubController) updateAuthorized(name string) error {
	if c.cacheClientset[name] == nil {
		return fmt.Errorf("hub %q is disconnected", name)
	}
	secret, err := c.cacheClientset[name].
		Kubernetes().
		CoreV1().
		Secrets(consts.FerryTunnelNamespace).
		Get(c.ctx, consts.FerryTunnelName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if secret.Data == nil {
		return fmt.Errorf("hub %q secret %s.%s is empty", name, consts.FerryTunnelName, consts.FerryTunnelNamespace)
	}
	authorized := secret.Data["authorized_keys"]
	if len(authorized) == 0 {
		return fmt.Errorf("hub %q not found authorized_keys key", name)
	}
	c.cacheAuthorized[name] = string(authorized)
	return nil
}

func (c *HubController) tryConnectAndUpdateStatus(name string) (clientset client.Interface, updated bool, err error) {
	old := c.cacheKubeconfig[name]
	err = c.updateKubeconfig(name)
	if err != nil {
		return nil, false, err
	}
	kubeconfig := c.cacheKubeconfig[name]
	if bytes.Equal(old, kubeconfig) {
		clientset = c.cacheClientset[name]
		if clientset != nil {
			// No need update
			return clientset, false, nil
		}
	}

	restConfig, err := client.NewRestConfigFromKubeconfig(kubeconfig)
	if err != nil {
		return nil, false, err
	}

	clientset, err = client.NewForConfig(restConfig)
	if err != nil {
		return nil, false, err
	}
	c.cacheClientset[name] = clientset

	err = c.updateAuthorized(name)
	if err != nil {
		return nil, false, err
	}
	return clientset, true, nil
}

func (c *HubController) updateKubeconfig(name string) error {
	secret, err := c.clientset.
		Kubernetes().
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
	f := newObj.(*trafficv1alpha2.Hub)
	f = f.DeepCopy()
	c.logger.Info("onUpdate",
		"hub", objref.KObj(f),
	)

	c.mut.Lock()
	c.cacheHub[f.Name] = f
	c.mut.Unlock()

	_, err := c.UpdateClientset(f.Name)
	if err != nil {
		c.logger.Error(err, "UpdateClientset",
			"hub", objref.KRef(consts.FerryNamespace, f.Name),
		)
	}
}

func (c *HubController) checkHealth(hubName string) {
	host := c.GetTunnelAddressInControlPlane(hubName)
	route := healthclient.NewClient("http://" + host)
	err := route.Get(c.ctx)
	if err != nil {
		c.logger.Error(err, "health",
			"hub", objref.KRef(consts.FerryNamespace, hubName),
		)
		c.UpdateHubConditions(hubName, []metav1.Condition{
			{
				Type:    trafficv1alpha2.TunnelHealthCondition,
				Status:  metav1.ConditionFalse,
				Reason:  "Unhealth",
				Message: err.Error(),
			},
		})
	} else {
		c.UpdateHubConditions(hubName, []metav1.Condition{
			{
				Type:   trafficv1alpha2.TunnelHealthCondition,
				Status: metav1.ConditionTrue,
				Reason: "Health",
			},
		})
	}
}

func (c *HubController) enablePorts(hubName string) {
	if c.cacheTunnelPorts[hubName] == nil {
		host := c.GetTunnelAddressInControlPlane(hubName)
		cli := portsclient.NewClient("http://" + host + "/ports")
		c.cacheTunnelPorts[hubName] = newTunnelPorts(tunnelPortsConfig{
			Logger: c.logger.WithName(hubName).WithName("tunnel-port"),
			GetUnusedPort: func() (int32, error) {
				return cli.Get(c.ctx)
			},
		})
	}
}

func (c *HubController) enableCache(hubName string, clientset client.Interface) {
	if clientset == nil {
		return
	}

	if c.cacheService[hubName] == nil {
		clusterService := newClusterServiceCache(clusterServiceCacheConfig{
			Clientset: clientset,
			Logger:    c.logger.WithName(hubName).WithName("service"),
			SyncFunc:  c.syncFunc,
		})
		c.cacheService[hubName] = clusterService
		err := clusterService.Start(c.ctx)
		if err != nil {
			c.logger.Error(err, "failed start cluster service cache")
		}
	} else {
		err := c.cacheService[hubName].ResetClientset(clientset)
		if err != nil {
			c.logger.Error(err, "reset clientset")
		}
	}
}

func (c *HubController) enableMCS(f *trafficv1alpha2.Hub, clientset client.Interface) {
	if clientset == nil {
		return
	}
	if c.cacheServiceExport[f.Name] == nil {
		clusterServiceExport := newClusterServiceExportCache(clusterServiceExportCacheConfig{
			Logger:    c.logger.WithName(f.Name).WithName("service-export"),
			Clientset: clientset,
			SyncFunc:  c.syncFunc,
		})
		err := clusterServiceExport.Start(c.ctx)
		if err != nil {
			c.logger.Error(err, "failed start cluster service exports cache")
		} else {
			c.cacheServiceExport[f.Name] = clusterServiceExport
		}
	} else {
		err := c.cacheServiceExport[f.Name].ResetClientset(clientset)
		if err != nil {
			c.logger.Error(err, "failed reset client")
		}
	}

	if c.cacheServiceImport[f.Name] == nil {
		clusterServiceImport := newClusterServiceImportCache(clusterServiceImportCacheConfig{
			Logger:    c.logger.WithName(f.Name).WithName("service-import"),
			Clientset: clientset,
			SyncFunc:  c.syncFunc,
		})
		err := clusterServiceImport.Start(c.ctx)
		if err != nil {
			c.logger.Error(err, "failed start cluster service imports cache")
		} else {
			c.cacheServiceImport[f.Name] = clusterServiceImport
		}
	} else {
		err := c.cacheServiceImport[f.Name].ResetClientset(clientset)
		if err != nil {
			c.logger.Error(err, "failed reset client")
		}
	}
	return
}

func (c *HubController) disableMCS(f *trafficv1alpha2.Hub) {
	if c.cacheServiceExport[f.Name] != nil {
		c.cacheServiceExport[f.Name].Close()
		delete(c.cacheServiceExport, f.Name)
	}
	if c.cacheServiceImport[f.Name] != nil {
		c.cacheServiceImport[f.Name].Close()
		delete(c.cacheServiceImport, f.Name)
	}
}

func (c *HubController) ListMCS(namespace string) (map[string][]*mcsv1alpha1.ServiceImport, map[string][]*mcsv1alpha1.ServiceExport) {
	c.mut.RLock()
	defer c.mut.RUnlock()
	if len(c.cacheServiceImport) == 0 && len(c.cacheServiceExport) == 0 {
		return nil, nil
	}

	importMap := map[string][]*mcsv1alpha1.ServiceImport{}
	for name, imports := range c.cacheServiceImport {
		list := imports.ListByNamespace(namespace)
		if len(list) == 0 {
			continue
		}
		importMap[name] = list
	}

	exportMap := map[string][]*mcsv1alpha1.ServiceExport{}
	for name, exports := range c.cacheServiceExport {
		list := exports.ListByNamespace(namespace)
		if len(list) == 0 {
			continue
		}
		exportMap[name] = list
	}
	return importMap, exportMap
}

func (c *HubController) onDelete(obj interface{}) {
	f := obj.(*trafficv1alpha2.Hub)
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
	delete(c.cacheAuthorized, f.Name)
	c.disableMCS(f)

	c.conditionsManager.Delete(f.Name)
	c.syncFunc()
}

func (c *HubController) GetHub(name string) *trafficv1alpha2.Hub {
	c.mut.RLock()
	defer c.mut.RUnlock()
	return c.cacheHub[name]
}

func (c *HubController) ListHubs() []*trafficv1alpha2.Hub {
	c.mut.RLock()
	defer c.mut.RUnlock()
	out := make([]*trafficv1alpha2.Hub, 0, len(c.cacheHub))
	for _, hub := range c.cacheHub {
		out = append(out, hub.DeepCopy())
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out
}

func (c *HubController) GetHubGateway(hubName string, forHub string) trafficv1alpha2.HubSpecGateway {
	hub := c.GetHub(hubName)
	if hub != nil {
		if hub.Spec.Override != nil {
			h, ok := hub.Spec.Override[forHub]
			if ok {
				return h
			}
		}
		return hub.Spec.Gateway
	}
	return trafficv1alpha2.HubSpecGateway{}
}

func (c *HubController) Sync(ctx context.Context) {
	hubs := c.ListHubs()
	for _, hub := range hubs {
		connectedCondition := c.conditionsManager.Find(hub.Name, trafficv1alpha2.ConnectedCondition)
		if connectedCondition == nil || (connectedCondition.Status == metav1.ConditionFalse &&
			time.Since(connectedCondition.LastTransitionTime.Time) > 10*time.Second) {
			c.checkHealth(hub.Name)
		}

		tunnelHealthCondition := c.conditionsManager.Find(hub.Name, trafficv1alpha2.TunnelHealthCondition)
		if tunnelHealthCondition == nil || (tunnelHealthCondition.Status == metav1.ConditionFalse &&
			time.Since(tunnelHealthCondition.LastTransitionTime.Time) > 10*time.Second) {
			c.ResetClientset(hub.Name)
		}
	}
}
