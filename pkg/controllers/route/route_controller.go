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
	"sort"
	"sync"

	trafficv1alpha2 "github.com/ferryproxy/api/apis/traffic/v1alpha2"
	externalversions "github.com/ferryproxy/client-go/generated/informers/externalversions"
	"github.com/ferryproxy/ferry/pkg/client"
	"github.com/ferryproxy/ferry/pkg/conditions"
	"github.com/ferryproxy/ferry/pkg/consts"
	"github.com/ferryproxy/ferry/pkg/utils/objref"
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
)

type RouteControllerConfig struct {
	Logger       logr.Logger
	Clientset    client.Interface
	HubInterface HubInterface
	Namespace    string
	SyncFunc     func()
}

type RouteController struct {
	ctx                    context.Context
	mut                    sync.RWMutex
	mutStatus              sync.Mutex
	clientset              client.Interface
	hubInterface           HubInterface
	cache                  map[string]*trafficv1alpha2.Route
	cacheMappingController map[clusterPair]*MappingController
	cacheRoutes            map[clusterPair][]*trafficv1alpha2.Route
	namespace              string
	syncFunc               func()
	logger                 logr.Logger
	conditionsManager      *conditions.ConditionsManager
}

func NewRouteController(conf *RouteControllerConfig) *RouteController {
	return &RouteController{
		clientset:              conf.Clientset,
		namespace:              conf.Namespace,
		hubInterface:           conf.HubInterface,
		logger:                 conf.Logger,
		syncFunc:               conf.SyncFunc,
		cache:                  map[string]*trafficv1alpha2.Route{},
		cacheMappingController: map[clusterPair]*MappingController{},
		cacheRoutes:            map[clusterPair][]*trafficv1alpha2.Route{},
		conditionsManager:      conditions.NewConditionsManager(),
	}
}

func (c *RouteController) list() []*trafficv1alpha2.Route {
	var list []*trafficv1alpha2.Route
	for _, v := range c.cache {
		item := c.cache[v.Name]
		if item == nil {
			continue
		}
		list = append(list, item)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].CreationTimestamp.Before(&list[j].CreationTimestamp)
	})
	return list
}

func (c *RouteController) Run(ctx context.Context) error {
	c.logger.Info("Route controller started")
	defer c.logger.Info("Route controller stopped")

	c.ctx = ctx
	informerFactory := externalversions.NewSharedInformerFactoryWithOptions(c.clientset.Ferry(), 0,
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

func (c *RouteController) UpdateRouteCondition(name string, conditions []metav1.Condition) {
	c.mutStatus.Lock()
	defer c.mutStatus.Unlock()

	var retErr error
	defer func() {
		if retErr != nil {
			c.logger.Error(retErr, "failed to update status")
		}
	}()

	fp := c.cache[name]
	if fp == nil {
		retErr = fmt.Errorf("not found route %s", name)
		return
	}

	status := fp.Status.DeepCopy()

	for _, condition := range conditions {
		c.conditionsManager.Set(name, condition)
	}

	ready, reason := c.conditionsManager.Ready(name,
		trafficv1alpha2.PortsAllocatedCondition,
		trafficv1alpha2.RouteSyncedCondition,
		trafficv1alpha2.ExportHubReadyCondition,
		trafficv1alpha2.ImportHubReadyCondition,
		trafficv1alpha2.PathReachableCondition,
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

	status.LastSynchronizationTimestamp = metav1.Now()
	status.Import = fmt.Sprintf("%s.%s", fp.Spec.Import.Service.Name, fp.Spec.Import.Service.Namespace)
	status.Export = fmt.Sprintf("%s.%s", fp.Spec.Export.Service.Name, fp.Spec.Export.Service.Namespace)
	status.Conditions = c.conditionsManager.Get(name)

	if cond := c.conditionsManager.Find(name, trafficv1alpha2.PathReachableCondition); cond != nil {
		if c.conditionsManager.IsTrue(name, trafficv1alpha2.PathReachableCondition) {
			status.Way = cond.Message
		} else {
			status.Way = "<unreachable>"
		}
	} else {
		status.Way = "<unknown>"
	}

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
		Routes(fp.Namespace).
		Patch(c.ctx, fp.Name, types.MergePatchType, data, metav1.PatchOptions{}, "status")
	if err != nil {
		retErr = err
		return
	}
}

func (c *RouteController) onAdd(obj interface{}) {
	f := obj.(*trafficv1alpha2.Route)
	f = f.DeepCopy()
	c.logger.Info("onAdd",
		"route", objref.KObj(f),
	)

	c.mut.Lock()
	defer c.mut.Unlock()

	c.cache[f.Name] = f

	c.syncFunc()
}

func (c *RouteController) onUpdate(oldObj, newObj interface{}) {
	f := newObj.(*trafficv1alpha2.Route)
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
}

func (c *RouteController) onDelete(obj interface{}) {
	f := obj.(*trafficv1alpha2.Route)
	c.logger.Info("onDelete",
		"route", objref.KObj(f),
	)

	c.mut.Lock()
	defer c.mut.Unlock()

	delete(c.cache, f.Name)

	c.conditionsManager.Delete(f.Name)
	c.syncFunc()
}

func (c *RouteController) Sync(ctx context.Context) {
	c.mut.Lock()
	defer c.mut.Unlock()

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

	err := mc.Start(ctx)
	if err != nil {
		return nil, err
	}

	c.cacheMappingController[key] = mc
	return mc, nil
}

func groupRoutes(rules []*trafficv1alpha2.Route) map[clusterPair][]*trafficv1alpha2.Route {
	mapping := map[clusterPair][]*trafficv1alpha2.Route{}

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
			mapping[key] = []*trafficv1alpha2.Route{}
		}

		mapping[key] = append(mapping[key], spec)
	}
	return mapping
}

type clusterPair struct {
	Export string
	Import string
}

func calculateRoutesPatch(older, newer map[clusterPair][]*trafficv1alpha2.Route) (updated, deleted []clusterPair) {
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
