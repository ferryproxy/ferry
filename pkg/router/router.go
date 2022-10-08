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
	"github.com/ferryproxy/ferry/pkg/router/discovery"
	"github.com/ferryproxy/ferry/pkg/utils/diffobjs"
	"github.com/ferryproxy/ferry/pkg/utils/maps"
	"github.com/ferryproxy/ferry/pkg/utils/objref"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type HubInterface interface {
	GetHubGateway(hubName string, forHub string) v1alpha2.HubSpecGateway
	GetAuthorized(name string) string
	GetPortPeer(importHubName string, cluster, namespace, name string, port int32) (int32, error)
}

type RouterConfig struct {
	Labels        map[string]string
	ExportHubName string
	ImportHubName string
	HubInterface  HubInterface
}

func NewRouter(conf RouterConfig) *Router {
	return &Router{
		labels:        conf.Labels,
		importHubName: conf.ImportHubName,
		exportHubName: conf.ExportHubName,
		hubInterface:  conf.HubInterface,
		hubsChain: NewHubsChain(HubsChainConfig{
			GetHubGateway: conf.HubInterface.GetHubGateway,
		}),
	}
}

type Router struct {
	labels map[string]string

	exportHubName string
	importHubName string

	hubInterface HubInterface

	hubsChain *HubsChain
}

func (d *Router) BuildResource(routes []*v1alpha2.Route, ways []string) (out map[string][]objref.KMetadata, err error) {
	out = map[string][]objref.KMetadata{}

	labelsForRules := maps.Merge(d.labels, map[string]string{
		consts.TunnelConfigKey: consts.TunnelConfigRulesValue,
	})
	labelsForAllow := maps.Merge(d.labels, map[string]string{
		consts.TunnelConfigKey: consts.TunnelConfigAllowValue,
	})
	labelsForAuth := maps.Merge(d.labels, map[string]string{
		consts.TunnelConfigKey: consts.TunnelConfigAuthorizedValue,
	})
	labelsForDiscover := maps.Merge(d.labels, map[string]string{
		consts.TunnelConfigKey: consts.TunnelConfigDiscoverValue,
	})

	for _, route := range routes {
		destination := objref.ObjectRef{Name: route.Spec.Import.Service.Name, Namespace: route.Spec.Import.Service.Namespace}
		origin := objref.ObjectRef{Name: route.Spec.Export.Service.Name, Namespace: route.Spec.Export.Service.Namespace}

		peerPortMapping := map[int32]int32{}

		exportPorts := route.Spec.Export.Ports
		importPorts := route.Spec.Import.Ports

		if len(exportPorts) != len(importPorts) {
			min := len(exportPorts)
			if len(importPorts) < min {
				min = len(importPorts)
			}
			exportPorts = exportPorts[:min]
			importPorts = importPorts[:min]
		}
		if len(exportPorts) == 0 {
			continue
		}

		for _, exportPort := range exportPorts {
			peerPort, err := d.hubInterface.GetPortPeer(d.importHubName, d.exportHubName, origin.Namespace, origin.Name, exportPort.Port)
			if err != nil {
				return nil, err
			}
			peerPortMapping[exportPort.Port] = peerPort

			tunnelName := fmt.Sprintf("%s-tunnel-%d-%d", route.Name, exportPort.Port, peerPort)
			hubsBound, err := d.hubsChain.Build(tunnelName, origin, destination, exportPort.Port, peerPort, ways)
			if err != nil {
				return nil, err
			}
			resources, err := ConvertOutboundToResourcers(tunnelName, consts.FerryTunnelNamespace, labelsForRules, hubsBound)
			if err != nil {
				return nil, err
			}
			for k, res := range resources {
				out[k] = append(out[k], res...)
			}

			allowName := fmt.Sprintf("%s-allows-%d-%d", route.Name, exportPort.Port, peerPort)
			resources, err = ConvertInboundToResourcers(allowName, consts.FerryTunnelNamespace, labelsForAllow, hubsBound)
			if err != nil {
				return nil, err
			}
			for k, res := range resources {
				out[k] = append(out[k], res...)
			}

			authNameSuffix := "authorized"
			resources, err = ConvertInboundAuthorizedToResourcers(authNameSuffix, consts.FerryTunnelNamespace, labelsForAuth, hubsBound, d.hubInterface.GetAuthorized)
			if err != nil {
				return nil, err
			}
			for k, res := range resources {
				out[k] = append(out[k], res...)
			}
		}

		serviceName := fmt.Sprintf("%s-service", route.Name)

		ports := buildPorts(peerPortMapping, exportPorts, importPorts)

		svcConfig := discovery.Service{
			ExportHubName:          d.exportHubName,
			ExportServiceNamespace: origin.Namespace,
			ExportServiceName:      origin.Name,
			ImportServiceNamespace: destination.Namespace,
			ImportServiceName:      destination.Name,
			Ports:                  ports,
		}
		data, err := svcConfig.ToMap()
		if err != nil {
			return nil, err
		}
		configMap := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      serviceName,
				Namespace: consts.FerryTunnelNamespace,
				Labels:    labelsForDiscover,
			},
			Data: data,
		}
		out[d.importHubName] = append(out[d.importHubName], configMap)

	}

	for name := range out {
		out[name] = diffobjs.Unique(out[name])
	}
	return out, nil
}

func buildPorts(peerPortMapping map[int32]int32, exportPorts, importPorts []v1alpha2.RouteSpecRulePort) []discovery.MappingPort {
	ports := []discovery.MappingPort{}
	for i, importPort := range importPorts {
		exportPort := exportPorts[i]
		svcPort := peerPortMapping[exportPort.Port]
		ports = append(ports, discovery.MappingPort{
			Name:       importPort.Name,
			Port:       importPort.Port,
			Protocol:   "TCP",
			TargetPort: svcPort,
		})
	}
	return ports
}
