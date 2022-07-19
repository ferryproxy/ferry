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

package router

import (
	"fmt"

	"github.com/ferryproxy/api/apis/traffic/v1alpha2"
	"github.com/ferryproxy/ferry/pkg/consts"
	"github.com/ferryproxy/ferry/pkg/ferry-controller/router/resource"
	"github.com/ferryproxy/ferry/pkg/utils/maps"
	"github.com/ferryproxy/ferry/pkg/utils/objref"
	corev1 "k8s.io/api/core/v1"
)

type RouterConfig struct {
	Namespace     string
	Labels        map[string]string
	ExportHubName string
	ImportHubName string
	GetIdentity   func(hubName string) string
	GetHubGateway func(hubName string, forHub string) v1alpha2.HubSpecGateway
	ListServices  func(name string) []*corev1.Service
	GetPortPeer   func(importHubName string, cluster, namespace, name string, port int32) int32
}

func NewRouter(conf RouterConfig) *Router {
	return &Router{
		namespace:     conf.Namespace,
		labels:        conf.Labels,
		importHubName: conf.ImportHubName,
		exportHubName: conf.ExportHubName,
		listServices:  conf.ListServices,
		getPortPeer:   conf.GetPortPeer,
		mappings:      map[objref.ObjectRef][]objref.ObjectRef{},
		hubsChain: NewHubsChain(HubsChainConfig{
			GetIdentity:   conf.GetIdentity,
			GetHubGateway: conf.GetHubGateway,
		}),
	}
}

type Router struct {
	namespace string
	labels    map[string]string

	exportHubName string
	importHubName string

	mappings map[objref.ObjectRef][]objref.ObjectRef

	listServices func(name string) []*corev1.Service
	getPortPeer  func(importHubName string, cluster, namespace, name string, port int32) int32

	hubsChain *HubsChain
}

func (d *Router) SetRoutes(rules []*v1alpha2.Route) {
	mappings := map[objref.ObjectRef][]objref.ObjectRef{}
	for _, rule := range rules {
		exportRef := objref.ObjectRef{Name: rule.Spec.Export.Service.Name, Namespace: rule.Spec.Export.Service.Namespace}
		importRef := objref.ObjectRef{Name: rule.Spec.Import.Service.Name, Namespace: rule.Spec.Import.Service.Namespace}
		mappings[exportRef] = append(mappings[exportRef], importRef)
	}
	d.mappings = mappings
}

func (d *Router) BuildResource(ways []string) (out map[string][]resource.Resourcer, err error) {
	out = map[string][]resource.Resourcer{}
	svcs := d.listServices(d.exportHubName)

	for _, svc := range svcs {
		origin := objref.KObj(svc)
		labels := maps.Merge(d.labels, map[string]string{
			consts.LabelFerryExportedFromNamespaceKey: origin.Namespace,
			consts.LabelFerryExportedFromNameKey:      origin.Name,
			consts.LabelFerryTunnelKey:                consts.LabelFerryTunnelValue,
		})
		for _, destination := range d.mappings[origin] {

			peerPortMapping := map[int32]int32{}
			for _, port := range svc.Spec.Ports {

				peerPort := d.getPortPeer(d.importHubName, d.exportHubName, destination.Namespace, destination.Name, port.Port)
				peerPortMapping[port.Port] = peerPort

				name := fmt.Sprintf("%s-%s-%s-%d-%s-%s-%s-%d-tunnel", d.importHubName, destination.Namespace, destination.Name, port.Port, d.exportHubName, origin.Namespace, origin.Name, peerPort)
				hubsChains, err := d.hubsChain.Build(name, origin, destination, port.Port, peerPort, ways)
				if err != nil {
					return nil, err
				}
				resources, err := ConvertChainsToResourcers(name, consts.FerryTunnelNamespace, labels, hubsChains)
				if err != nil {
					return nil, err
				}
				for k, res := range resources {
					out[k] = append(out[k], res...)
				}
			}

			resources, err := BuildServiceDiscovery(destination.Name, destination.Namespace, labels, peerPortMapping, &svc.Spec)
			if err != nil {
				return nil, err
			}
			out[d.importHubName] = append(out[d.importHubName], resources...)
		}
	}

	return out, nil
}
