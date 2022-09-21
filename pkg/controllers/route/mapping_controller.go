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
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/ferryproxy/api/apis/traffic/v1alpha2"
	"github.com/ferryproxy/ferry/pkg/client"
	"github.com/ferryproxy/ferry/pkg/consts"
	"github.com/ferryproxy/ferry/pkg/router"
	"github.com/ferryproxy/ferry/pkg/router/discovery"
	"github.com/ferryproxy/ferry/pkg/utils/diffobjs"
	"github.com/ferryproxy/ferry/pkg/utils/objref"
	"github.com/ferryproxy/ferry/pkg/utils/trybuffer"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

type HubInterface interface {
	GetService(hubName string, namespace, name string) (*corev1.Service, bool)
	ListServices(name string) []*corev1.Service
	GetHub(name string) *v1alpha2.Hub
	GetHubGateway(hubName string, forHub string) v1alpha2.HubSpecGateway
	GetAuthorized(name string) string
	Clientset(name string) client.Interface
	LoadPortPeer(importHubName string, cluster, namespace, name string, port, bindPort int32) error
	GetPortPeer(importHubName string, cluster, namespace, name string, port int32) (int32, error)
	DeletePortPeer(importHubName string, cluster, namespace, name string, port int32) (int32, error)
	RegistryServiceCallback(exportHubName, importHubName string, cb func())
	UnregistryServiceCallback(exportHubName, importHubName string)
	HubReady(hubName string) bool
}

type RouteInterface interface {
	UpdateRouteCondition(name string, conditions []metav1.Condition) error
}

type MappingControllerConfig struct {
	Namespace      string
	ExportHubName  string
	ImportHubName  string
	HubInterface   HubInterface
	RouteInterface RouteInterface
	Logger         logr.Logger
}

func NewMappingController(conf MappingControllerConfig) *MappingController {
	return &MappingController{
		namespace:      conf.Namespace,
		importHubName:  conf.ImportHubName,
		exportHubName:  conf.ExportHubName,
		logger:         conf.Logger,
		hubInterface:   conf.HubInterface,
		routeInterface: conf.RouteInterface,
		cacheResources: map[string][]objref.KMetadata{},
	}
}

type MappingController struct {
	mut sync.Mutex
	ctx context.Context

	namespace string
	labels    map[string]string

	exportHubName string
	importHubName string

	router         *router.Router
	solution       *router.Solution
	hubInterface   HubInterface
	routeInterface RouteInterface

	routes         []*v1alpha2.Route
	cacheResources map[string][]objref.KMetadata
	logger         logr.Logger
	way            []string

	try *trybuffer.TryBuffer

	isClose bool
}

func (d *MappingController) Start(ctx context.Context) error {
	d.mut.Lock()
	defer d.mut.Unlock()

	d.logger.Info("DataPlane controller started")
	defer func() {
		d.logger.Info("DataPlane controller stopped")
	}()
	d.ctx = ctx

	d.solution = router.NewSolution(router.SolutionConfig{
		GetHubGateway: d.hubInterface.GetHubGateway,
	})

	// Mark managed by ferry
	opt := metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(d.getLabel()).String(),
	}

	way, err := d.solution.CalculateWays(d.exportHubName, d.importHubName)
	if err != nil {
		d.logger.Error(err, "calculate ways")
		return err
	}
	d.way = way

	for _, w := range way {
		err := d.loadLastConfigMap(ctx, w, opt)
		if err != nil {
			return err
		}
	}
	d.router = router.NewRouter(router.RouterConfig{
		Labels:        d.getLabel(),
		ExportHubName: d.exportHubName,
		ImportHubName: d.importHubName,
		HubInterface:  d.hubInterface,
	})

	d.try = trybuffer.NewTryBuffer(d.sync, time.Second/10)

	d.hubInterface.RegistryServiceCallback(d.exportHubName, d.importHubName, d.Sync)

	return nil
}

func (d *MappingController) Sync() {
	d.try.Try()
}

func (d *MappingController) SetRoutes(routes []*v1alpha2.Route) {
	d.mut.Lock()
	defer d.mut.Unlock()
	if reflect.DeepEqual(d.routes, routes) {
		return
	}
	for _, route := range routes {
		conds := []metav1.Condition{}
		err := d.updatePort(route)
		if err != nil {
			conds = append(conds, metav1.Condition{
				Type:    v1alpha2.PortsAllocatedCondition,
				Status:  metav1.ConditionFalse,
				Reason:  "FailedPortsAllocated",
				Message: err.Error(),
			})
		} else {
			conds = append(conds, metav1.Condition{
				Type:   v1alpha2.PortsAllocatedCondition,
				Status: metav1.ConditionTrue,
				Reason: v1alpha2.PortsAllocatedCondition,
			})
		}
		err = d.routeInterface.UpdateRouteCondition(route.Name, conds)
		if err != nil {
			d.logger.Error(err, "failed to update status")
		}
	}
	deleted := diffobjs.ShouldDeleted(d.routes, routes)
	for _, route := range deleted {
		err := d.deletePort(route)
		if err != nil {
			d.logger.Error(err, "delete port")
		}
	}
	d.routes = routes
}

func (d *MappingController) loadLastConfigMap(ctx context.Context, name string, opt metav1.ListOptions) error {
	cmList, err := d.hubInterface.Clientset(name).
		Kubernetes().
		CoreV1().
		ConfigMaps(d.namespace).
		List(ctx, opt)
	if err != nil {
		return err
	}
	for _, item := range cmList.Items {
		d.cacheResources[name] = append(d.cacheResources[name], item.DeepCopy())
	}
	for _, item := range cmList.Items {
		if item.Labels != nil && item.Labels[consts.TunnelConfigKey] == consts.TunnelConfigDiscoverValue {
			d.loadPorts(name, &item)
		}
	}
	return nil
}

func (d *MappingController) loadPorts(importHubName string, cm *corev1.ConfigMap) {
	data, err := discovery.ServiceFrom(cm.Data)
	if err != nil {
		d.logger.Error(err, "ServiceFrom")
		return
	}
	for _, port := range data.Ports {
		err = d.hubInterface.LoadPortPeer(importHubName, data.ExportHubName, data.ExportServiceNamespace, data.ExportServiceName, port.Port, port.TargetPort)
		if err != nil {
			d.logger.Error(err, "LoadPortPeer")
		}
	}
}

func (d *MappingController) getLabel() map[string]string {
	if d.labels != nil {
		return d.labels
	}
	d.labels = map[string]string{
		consts.LabelGeneratedKey:         consts.LabelGeneratedValue,
		consts.LabelFerryExportedFromKey: d.exportHubName,
		consts.LabelFerryImportedToKey:   d.importHubName,
	}
	return d.labels
}

func (d *MappingController) sync() {
	d.mut.Lock()
	defer d.mut.Unlock()

	if d.isClose {
		return
	}
	ctx := d.ctx

	// TODO: check for failures sync
	conds := []metav1.Condition{}

	importHubReady := d.hubInterface.HubReady(d.importHubName)
	if importHubReady {
		conds = append(conds, metav1.Condition{
			Type:   v1alpha2.ImportHubReadyCondition,
			Status: metav1.ConditionTrue,
			Reason: v1alpha2.ImportHubReadyCondition,
		})
	} else {
		conds = append(conds, metav1.Condition{
			Type:   v1alpha2.ImportHubReadyCondition,
			Status: metav1.ConditionFalse,
			Reason: "NotReady",
		})
	}

	exportHubReady := d.hubInterface.HubReady(d.exportHubName)
	if exportHubReady {
		conds = append(conds, metav1.Condition{
			Type:   v1alpha2.ExportHubReadyCondition,
			Status: metav1.ConditionTrue,
			Reason: v1alpha2.ExportHubReadyCondition,
		})
	} else {
		conds = append(conds, metav1.Condition{
			Type:   v1alpha2.ExportHubReadyCondition,
			Status: metav1.ConditionFalse,
			Reason: "NotReady",
		})
	}

	defer func() {
		for _, route := range d.routes {
			err := d.routeInterface.UpdateRouteCondition(route.Name, conds)
			if err != nil {
				d.logger.Error(err, "failed to update status")
			}
		}
	}()

	way, err := d.solution.CalculateWays(d.exportHubName, d.importHubName)
	if err != nil {
		d.logger.Error(err, "calculate ways")
		return
	}

	resources, err := d.router.BuildResource(d.routes, way)
	if err != nil {
		conds = append(conds,
			metav1.Condition{
				Type:    v1alpha2.PathReachableCondition,
				Status:  metav1.ConditionFalse,
				Reason:  "Unreachable",
				Message: err.Error(),
			},
		)
		d.logger.Error(err, "build resource")
		return
	}
	msg := ""
	if len(way) == 2 {
		msg = "<Direct>"
	} else {
		msg = strings.Join(way[1:len(way)-1], ",")
	}

	conds = append(conds,
		metav1.Condition{
			Type:    v1alpha2.PathReachableCondition,
			Status:  metav1.ConditionTrue,
			Reason:  v1alpha2.PathReachableCondition,
			Message: msg,
		},
	)

	d.way = way

	defer func() {
		d.cacheResources = resources
	}()

	for hubName, updated := range resources {
		cacheResource := d.cacheResources[hubName]
		deleled := diffobjs.ShouldDeleted(cacheResource, updated)
		cli := d.hubInterface.Clientset(hubName)
		for _, r := range updated {
			err := client.Apply(ctx, cli, r)
			if err != nil {
				d.logger.Error(err, "Apply resource",
					"hub", objref.KRef(consts.FerryNamespace, hubName),
				)
			}
		}

		for _, r := range deleled {
			err := client.Delete(ctx, cli, r)
			if err != nil {
				d.logger.Error(err, "Delete resource",
					"hub", objref.KRef(consts.FerryNamespace, hubName),
				)
			}
		}
	}

	for hubName, caches := range d.cacheResources {
		v, ok := resources[hubName]
		if ok && len(v) != 0 {
			continue
		}
		cli := d.hubInterface.Clientset(hubName)
		for _, r := range caches {
			err := client.Apply(ctx, cli, r)
			if err != nil {
				d.logger.Error(err, "Delete resource",
					"hub", objref.KRef(consts.FerryNamespace, hubName),
				)
			}
		}
	}

	conds = append(conds,
		metav1.Condition{
			Type:   v1alpha2.RouteSyncedCondition,
			Status: metav1.ConditionTrue,
			Reason: v1alpha2.RouteSyncedCondition,
		},
	)

	return
}

func (d *MappingController) Close() {
	d.mut.Lock()
	defer d.mut.Unlock()

	if d.isClose {
		return
	}
	d.isClose = true
	d.hubInterface.UnregistryServiceCallback(d.exportHubName, d.importHubName)
	d.try.Close()

	ctx := context.Background()

	for hubName, caches := range d.cacheResources {
		cli := d.hubInterface.Clientset(hubName)
		for _, r := range caches {
			err := client.Delete(ctx, cli, r)
			if err != nil {
				d.logger.Error(err, "Delete resource",
					"hub", objref.KRef(consts.FerryNamespace, hubName),
				)
			}
		}
	}
}

func (c *MappingController) updatePort(f *v1alpha2.Route) error {
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

func (c *MappingController) deletePort(f *v1alpha2.Route) error {
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
