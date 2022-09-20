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

package route

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sync"

	"github.com/ferryproxy/api/apis/traffic/v1alpha2"
	versioned "github.com/ferryproxy/client-go/generated/clientset/versioned"
	externalversions "github.com/ferryproxy/client-go/generated/informers/externalversions"
	"github.com/ferryproxy/ferry/pkg/conditions"
	"github.com/ferryproxy/ferry/pkg/consts"
	"github.com/ferryproxy/ferry/pkg/utils/objref"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

type RouteControllerConfig struct {
	Logger       logr.Logger
	Config       *restclient.Config
	HubInterface HubInterface
	Namespace    string
	SyncFunc     func()
}

type RouteController struct {
	ctx                    context.Context
	mut                    sync.RWMutex
	mutStatus              sync.Mutex
	config                 *restclient.Config
	clientset              versioned.Interface
	hubInterface           HubInterface
	cache                  map[string]*v1alpha2.Route
	cacheMappingController map[clusterPair]*MappingController
	cacheRoutes            map[clusterPair][]*v1alpha2.Route
	namespace              string
	syncFunc               func()
	logger                 logr.Logger
	conditionsManager      *conditions.ConditionsManager
}

func NewRouteController(conf *RouteControllerConfig) *RouteController {
	return &RouteController{
		config:                 conf.Config,
		namespace:              conf.Namespace,
		hubInterface:           conf.HubInterface,
		logger:                 conf.Logger,
		syncFunc:               conf.SyncFunc,
		cache:                  map[string]*v1alpha2.Route{},
		cacheMappingController: map[clusterPair]*MappingController{},
		cacheRoutes:            map[clusterPair][]*v1alpha2.Route{},
		conditionsManager:      conditions.NewConditionsManager(),
	}
}

func (c *RouteController) list() []*v1alpha2.Route {
	var list []*v1alpha2.Route
	for _, v := range c.cache {
		item := c.cache[v.Name]
		if item == nil {
			continue
		}
		list = append(list, item)
	}
	return list
}

func (c *RouteController) Run(ctx context.Context) error {
	c.logger.Info("Route controller started")
	defer c.logger.Info("Route controller stopped")

	clientset, err := versioned.NewForConfig(c.config)
	if err != nil {
		return err
	}
	c.clientset = clientset
	c.ctx = ctx
	informerFactory := externalversions.NewSharedInformerFactoryWithOptions(clientset, 0,
		externalversions.WithNamespace(c.namespace))
	informer := informerFactory.
		Traffic().
		V1alpha2().
		Routes().
		Informer()
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.onAdd,
		UpdateFunc: c.onUpdate,
		DeleteFunc: c.onDelete,
	})

	informer.Run(ctx.Done())
	return nil
}

func (c *RouteController) UpdateRouteCondition(name string, conditions []metav1.Condition) error {
	c.mutStatus.Lock()
	defer c.mutStatus.Unlock()
	fp := c.cache[name]
	if fp == nil {
		return fmt.Errorf("not found Route %s", name)
	}

	status := fp.Status.DeepCopy()

	for _, condition := range conditions {
		c.conditionsManager.Set(name, condition)
	}

	if c.conditionsManager.IsTrue(name, v1alpha2.PortsAllocatedCondition) &&
		c.conditionsManager.IsTrue(name, v1alpha2.RouteSyncedCondition) &&
		c.conditionsManager.IsTrue(name, v1alpha2.ExportHubReadyCondition) &&
		c.conditionsManager.IsTrue(name, v1alpha2.ImportHubReadyCondition) &&
		c.conditionsManager.IsTrue(name, v1alpha2.PathReachableCondition) {
		c.conditionsManager.Set(name, metav1.Condition{
			Type:   v1alpha2.RouteReady,
			Status: metav1.ConditionTrue,
			Reason: v1alpha2.RouteReady,
		})
		status.Phase = v1alpha2.RouteReady
	} else {
		c.conditionsManager.Set(name, metav1.Condition{
			Type:   v1alpha2.RouteReady,
			Status: metav1.ConditionFalse,
			Reason: "NotReady",
		})
		status.Phase = "NotReady"
	}

	status.LastSynchronizationTimestamp = metav1.Now()
	status.Import = fmt.Sprintf("%s.%s/%s", fp.Spec.Import.Service.Name, fp.Spec.Import.Service.Namespace, fp.Spec.Import.HubName)
	status.Export = fmt.Sprintf("%s.%s/%s", fp.Spec.Export.Service.Name, fp.Spec.Export.Service.Namespace, fp.Spec.Export.HubName)
	status.Conditions = c.conditionsManager.Get(name)

	if cond := c.conditionsManager.Find(name, v1alpha2.PathReachableCondition); cond != nil {
		if c.conditionsManager.IsTrue(name, v1alpha2.PathReachableCondition) {
			status.Way = cond.Message
		} else {
			status.Way = "<Unreachable>"
		}
	} else {
		status.Way = "<Unknown>"
	}

	data, err := json.Marshal(map[string]interface{}{
		"status": status,
	})
	if err != nil {
		return err
	}
	_, err = c.clientset.
		TrafficV1alpha2().
		Routes(fp.Namespace).
		Patch(c.ctx, fp.Name, types.MergePatchType, data, metav1.PatchOptions{}, "status")
	return err
}

func (c *RouteController) onAdd(obj interface{}) {
	f := obj.(*v1alpha2.Route)
	f = f.DeepCopy()
	c.logger.Info("onAdd",
		"route", objref.KObj(f),
	)

	c.mut.Lock()
	defer c.mut.Unlock()

	c.cache[f.Name] = f

	c.syncFunc()

	err := c.updatePort(f)
	if err != nil {
		err := c.UpdateRouteCondition(f.Name, []metav1.Condition{
			{
				Type:    v1alpha2.PortsAllocatedCondition,
				Status:  metav1.ConditionFalse,
				Reason:  "FailedPortsAllocated",
				Message: err.Error(),
			},
		})
		if err != nil {
			c.logger.Error(err, "failed to update status", "route", f.Name)
		}
		return
	}

	err = c.UpdateRouteCondition(f.Name, []metav1.Condition{
		{
			Type:   v1alpha2.PortsAllocatedCondition,
			Status: metav1.ConditionTrue,
			Reason: v1alpha2.PortsAllocatedCondition,
		},
	})
	if err != nil {
		c.logger.Error(err, "failed to update status", "route", f.Name)
	}
}

func (c *RouteController) updatePort(f *v1alpha2.Route) error {
	svc, ok := c.hubInterface.GetService(f.Spec.Export.HubName, f.Spec.Export.Service.Namespace, f.Spec.Export.Service.Name)
	if !ok {
		return fmt.Errorf("not found export service")
	}

	for _, port := range svc.Spec.Ports {
		if port.Protocol != corev1.ProtocolTCP {
			continue
		}
		_, err := c.hubInterface.GetPortPeer(f.Spec.Import.HubName,
			f.Spec.Export.HubName, f.Spec.Export.Service.Namespace, f.Spec.Export.Service.Name, port.Port)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *RouteController) deletePort(f *v1alpha2.Route) error {
	svc, ok := c.hubInterface.GetService(f.Spec.Export.HubName, f.Spec.Export.Service.Namespace, f.Spec.Export.Service.Name)
	if !ok {
		return fmt.Errorf("not found export service")
	}
	for _, port := range svc.Spec.Ports {
		if port.Protocol != corev1.ProtocolTCP {
			continue
		}
		_, err := c.hubInterface.DeletePortPeer(f.Spec.Import.HubName,
			f.Spec.Export.HubName, f.Spec.Export.Service.Namespace, f.Spec.Export.Service.Name, port.Port)
		if err == nil {
			continue
		}
	}
	return nil
}

func (c *RouteController) onUpdate(oldObj, newObj interface{}) {
	f := newObj.(*v1alpha2.Route)
	f = f.DeepCopy()
	c.logger.Info("onUpdate",
		"route", objref.KObj(f),
	)

	c.mut.Lock()
	defer c.mut.Unlock()

	if reflect.DeepEqual(c.cache[f.Name].Spec, f.Spec) {
		c.cache[f.Name] = f
		return
	}

	c.cache[f.Name] = f

	c.syncFunc()

	err := c.updatePort(f)
	if err != nil {
		err := c.UpdateRouteCondition(f.Name, []metav1.Condition{
			{
				Type:    v1alpha2.PortsAllocatedCondition,
				Status:  metav1.ConditionFalse,
				Reason:  "FailedPortsAllocated",
				Message: err.Error(),
			},
		})
		if err != nil {
			c.logger.Error(err, "failed to update status", "route", f.Name)
		}
		return
	}

	err = c.UpdateRouteCondition(f.Name, []metav1.Condition{
		{
			Type:   v1alpha2.PortsAllocatedCondition,
			Status: metav1.ConditionTrue,
			Reason: v1alpha2.PortsAllocatedCondition,
		},
	})
	if err != nil {
		c.logger.Error(err, "failed to update status", "route", f.Name)
	}
}

func (c *RouteController) onDelete(obj interface{}) {
	f := obj.(*v1alpha2.Route)
	c.logger.Info("onDelete",
		"route", objref.KObj(f),
	)

	c.mut.Lock()
	defer c.mut.Unlock()

	c.updatePort(f)

	delete(c.cache, f.Name)

	c.syncFunc()
}

func (c *RouteController) Sync(ctx context.Context) {
	c.mut.RLock()
	defer c.mut.RUnlock()

	routes := c.list()

	newerRoutes := groupRoutes(routes)
	defer func() {
		c.cacheRoutes = newerRoutes
	}()
	logger := c.logger.WithName("sync")

	updated, deleted := calculateRoutesPatch(c.cacheRoutes, newerRoutes)

	for _, key := range deleted {
		logger := logger.WithValues("export", key.Export, "import", key.Import)
		logger.Info("Delete mapping controller")
		c.cleanupMappingController(key)
	}

	for _, key := range updated {
		logger := logger.WithValues("export", key.Export, "import", key.Import)
		logger.Info("Update mapping controller")
		mc, err := c.startMappingController(ctx, key)
		if err != nil {
			logger.Error(err, "start mapping controller")
			continue
		}

		mc.SetRoutes(newerRoutes[key])

		mc.Sync()
	}
	return
}

func (c *RouteController) cleanupMappingController(key clusterPair) {
	mc := c.cacheMappingController[key]
	if mc != nil {
		mc.Close()
		delete(c.cacheMappingController, key)
	}
}

func (c *RouteController) getMappingController(key clusterPair) *MappingController {
	return c.cacheMappingController[key]
}

func (c *RouteController) startMappingController(ctx context.Context, key clusterPair) (*MappingController, error) {
	mc := c.cacheMappingController[key]
	if mc != nil {
		return mc, nil
	}

	exportClientset := c.hubInterface.Clientset(key.Export)
	if exportClientset == nil {
		return nil, fmt.Errorf("not found clientset %q", key.Export)
	}
	importClientset := c.hubInterface.Clientset(key.Import)
	if importClientset == nil {
		return nil, fmt.Errorf("not found clientset %q", key.Import)
	}

	exportCluster := c.hubInterface.GetHub(key.Export)
	if exportCluster == nil {
		return nil, fmt.Errorf("not found cluster information %q", key.Export)
	}

	importCluster := c.hubInterface.GetHub(key.Import)
	if importCluster == nil {
		return nil, fmt.Errorf("not found cluster information %q", key.Import)
	}

	mc = NewMappingController(MappingControllerConfig{
		Namespace:      consts.FerryTunnelNamespace,
		HubInterface:   c.hubInterface,
		RouteInterface: c,
		ImportHubName:  key.Import,
		ExportHubName:  key.Export,
		Logger: c.logger.WithName("data-plane").
			WithName(key.Import).
			WithValues("export", key.Export, "import", key.Import),
	})
	c.cacheMappingController[key] = mc

	err := mc.Start(ctx)
	if err != nil {
		return nil, err
	}
	return mc, nil
}

func groupRoutes(rules []*v1alpha2.Route) map[clusterPair][]*v1alpha2.Route {
	mapping := map[clusterPair][]*v1alpha2.Route{}

	for _, spec := range rules {
		rule := spec.Spec
		export := rule.Export
		impor := rule.Import

		if export.HubName == "" || impor.HubName == "" || impor.HubName == export.HubName {
			continue
		}

		key := clusterPair{
			Export: rule.Export.HubName,
			Import: rule.Import.HubName,
		}

		if _, ok := mapping[key]; !ok {
			mapping[key] = []*v1alpha2.Route{}
		}

		mapping[key] = append(mapping[key], spec)
	}
	return mapping
}

type clusterPair struct {
	Export string
	Import string
}

func calculateRoutesPatch(older, newer map[clusterPair][]*v1alpha2.Route) (updated, deleted []clusterPair) {
	exist := map[clusterPair]struct{}{}

	for key := range older {
		exist[key] = struct{}{}
	}

	for key := range newer {
		updated = append(updated, key)
		delete(exist, key)
	}

	for r := range exist {
		deleted = append(deleted, r)
	}
	return updated, deleted
}
