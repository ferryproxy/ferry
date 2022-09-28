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
	Clientset(hubName string) (client.Interface, error)
	LoadPortPeer(importHubName string, cluster, namespace, name string, port, bindPort int32) error
	GetPortPeer(importHubName string, cluster, namespace, name string, port int32) (int32, error)
	DeletePortPeer(importHubName string, cluster, namespace, name string, port int32) (int32, error)
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

func (m *MappingController) Start(ctx context.Context) error {
	m.mut.Lock()
	defer m.mut.Unlock()

	m.ctx = ctx

	m.solution = router.NewSolution(router.SolutionConfig{
		GetHubGateway: m.hubInterface.GetHubGateway,
	})

	// Mark managed by ferry
	opt := metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(m.getLabel()).String(),
	}

	way, err := m.solution.CalculateWays(m.exportHubName, m.importHubName)
	if err != nil {
		m.logger.Error(err, "calculate ways")
		return err
	}
	m.way = way

	for _, w := range way {
		err := m.loadLastConfigMap(ctx, w, opt)
		if err != nil {
			return err
		}
	}
	m.router = router.NewRouter(router.RouterConfig{
		Labels:        m.getLabel(),
		ExportHubName: m.exportHubName,
		ImportHubName: m.importHubName,
		HubInterface:  m.hubInterface,
	})

	m.try = trybuffer.NewTryBuffer(m.sync, time.Second/10)
	return nil
}

func (m *MappingController) Sync() {
	m.try.Try()
}

func (m *MappingController) SetRoutes(routes []*v1alpha2.Route) {
	m.mut.Lock()
	defer m.mut.Unlock()

	for _, route := range routes {
		conds := []metav1.Condition{}
		err := m.updatePort(route)
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
		err = m.routeInterface.UpdateRouteCondition(route.Name, conds)
		if err != nil {
			m.logger.Error(err, "failed to update status")
		}
	}
	deleted := diffobjs.ShouldDeleted(m.routes, routes)
	for _, route := range deleted {
		err := m.deletePort(route)
		if err != nil {
			m.logger.Error(err, "delete port")
		}
	}
	m.routes = routes
}

func (m *MappingController) loadLastConfigMap(ctx context.Context, name string, opt metav1.ListOptions) error {
	clientset, err := m.hubInterface.Clientset(name)
	if err != nil {
		return err
	}
	cmList, err := clientset.
		Kubernetes().
		CoreV1().
		ConfigMaps(m.namespace).
		List(ctx, opt)
	if err != nil {
		return err
	}
	for _, item := range cmList.Items {
		m.cacheResources[name] = append(m.cacheResources[name], item.DeepCopy())
	}
	for _, item := range cmList.Items {
		if item.Labels != nil && item.Labels[consts.TunnelConfigKey] == consts.TunnelConfigDiscoverValue {
			m.loadPorts(name, &item)
		}
	}
	return nil
}

func (m *MappingController) loadPorts(importHubName string, cm *corev1.ConfigMap) {
	data, err := discovery.ServiceFrom(cm.Data)
	if err != nil {
		m.logger.Error(err, "ServiceFrom")
		return
	}
	for _, port := range data.Ports {
		err = m.hubInterface.LoadPortPeer(importHubName, data.ExportHubName, data.ExportServiceNamespace, data.ExportServiceName, port.Port, port.TargetPort)
		if err != nil {
			m.logger.Error(err, "LoadPortPeer")
		}
	}
}

func (m *MappingController) getLabel() map[string]string {
	if m.labels != nil {
		return m.labels
	}
	m.labels = map[string]string{
		consts.LabelGeneratedKey:         consts.LabelGeneratedValue,
		consts.LabelFerryExportedFromKey: m.exportHubName,
		consts.LabelFerryImportedToKey:   m.importHubName,
	}
	return m.labels
}

func (m *MappingController) sync() {
	m.mut.Lock()
	defer m.mut.Unlock()

	if m.isClose {
		return
	}
	ctx := m.ctx

	conds := []metav1.Condition{}

	defer func() {
		if len(conds) != 0 {
			for _, route := range m.routes {
				err := m.routeInterface.UpdateRouteCondition(route.Name, conds)
				if err != nil {
					m.logger.Error(err, "failed to update status")
				}
			}
		}
	}()

	way, err := m.solution.CalculateWays(m.exportHubName, m.importHubName)
	if err != nil {
		conds = append(conds,
			metav1.Condition{
				Type:    v1alpha2.PathReachableCondition,
				Status:  metav1.ConditionFalse,
				Reason:  "Unreachable",
				Message: err.Error(),
			},
		)
		m.logger.Error(err, "calculate ways")
		return
	}

	resources, err := m.router.BuildResource(m.routes, way)
	if err != nil {
		conds = append(conds,
			metav1.Condition{
				Type:    v1alpha2.PathReachableCondition,
				Status:  metav1.ConditionFalse,
				Reason:  "Unreachable",
				Message: err.Error(),
			},
		)
		m.logger.Error(err, "build resource")
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

	m.way = way

	defer func() {
		m.cacheResources = resources
	}()

	for hubName, updated := range resources {
		cacheResource := m.cacheResources[hubName]
		deleled := diffobjs.ShouldDeleted(cacheResource, updated)
		cli, err := m.hubInterface.Clientset(hubName)
		if err != nil {
			m.logger.Error(err, "Clientset",
				"hub", objref.KRef(consts.FerryNamespace, hubName),
			)
			continue
		}
		for _, r := range updated {
			err := client.Apply(ctx, cli, r)
			if err != nil {
				m.logger.Error(err, "Apply resource",
					"hub", objref.KRef(consts.FerryNamespace, hubName),
				)
			}
		}

		for _, r := range deleled {
			err := client.Delete(ctx, cli, r)
			if err != nil {
				m.logger.Error(err, "Delete resource",
					"hub", objref.KRef(consts.FerryNamespace, hubName),
				)
			}
		}
	}

	for hubName, caches := range m.cacheResources {
		v, ok := resources[hubName]
		if ok && len(v) != 0 {
			continue
		}
		cli, err := m.hubInterface.Clientset(hubName)
		if err != nil {
			m.logger.Error(err, "Clientset",
				"hub", objref.KRef(consts.FerryNamespace, hubName),
			)
			continue
		}
		for _, r := range caches {
			err := client.Delete(ctx, cli, r)
			if err != nil {
				m.logger.Error(err, "Delete resource",
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

	importHubReady := m.hubInterface.HubReady(m.importHubName)
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

	exportHubReady := m.hubInterface.HubReady(m.exportHubName)
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

	return
}

func (m *MappingController) Close() {
	m.mut.Lock()
	defer m.mut.Unlock()

	if m.isClose {
		return
	}
	m.isClose = true
	m.try.Close()

	ctx := context.Background()

	for hubName, caches := range m.cacheResources {
		cli, err := m.hubInterface.Clientset(hubName)
		if err != nil {
			m.logger.Error(err, "Clientset",
				"hub", objref.KRef(consts.FerryNamespace, hubName),
			)
			continue
		}
		for _, r := range caches {
			err := client.Delete(ctx, cli, r)
			if err != nil {
				m.logger.Error(err, "Delete resource",
					"hub", objref.KRef(consts.FerryNamespace, hubName),
				)
			}
		}
	}
}

func (m *MappingController) updatePort(f *v1alpha2.Route) error {
	svc, ok := m.hubInterface.GetService(f.Spec.Export.HubName, f.Spec.Export.Service.Namespace, f.Spec.Export.Service.Name)
	if !ok {
		return fmt.Errorf("not found export service")
	}

	for _, port := range svc.Spec.Ports {
		if port.Protocol != corev1.ProtocolTCP {
			continue
		}
		_, err := m.hubInterface.GetPortPeer(f.Spec.Import.HubName,
			f.Spec.Export.HubName, f.Spec.Export.Service.Namespace, f.Spec.Export.Service.Name, port.Port)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *MappingController) deletePort(f *v1alpha2.Route) error {
	svc, ok := m.hubInterface.GetService(f.Spec.Export.HubName, f.Spec.Export.Service.Namespace, f.Spec.Export.Service.Name)
	if !ok {
		return fmt.Errorf("not found export service")
	}
	for _, port := range svc.Spec.Ports {
		if port.Protocol != corev1.ProtocolTCP {
			continue
		}
		_, err := m.hubInterface.DeletePortPeer(f.Spec.Import.HubName,
			f.Spec.Export.HubName, f.Spec.Export.Service.Namespace, f.Spec.Export.Service.Name, port.Port)
		if err == nil {
			continue
		}
	}
	return nil
}
