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

	trafficv1alpha2 "github.com/ferryproxy/api/apis/traffic/v1alpha2"
	"github.com/ferryproxy/ferry/pkg/consts"
	"github.com/ferryproxy/ferry/pkg/router/discovery"
	"github.com/ferryproxy/ferry/pkg/utils/diffobjs"
	"github.com/ferryproxy/ferry/pkg/utils/maps"
	"github.com/ferryproxy/ferry/pkg/utils/objref"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type HubInterface interface {
	ListServices(name string) []*corev1.Service
	GetHubGateway(hubName string, forHub string) trafficv1alpha2.HubSpecGateway
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

func (d *Router) BuildResource(rules []*trafficv1alpha2.Route, ways []string) (out map[string][]objref.KMetadata, err error) {
	mappings := map[objref.ObjectRef][]*trafficv1alpha2.Route{}

	for _, rule := range rules {
		exportRef := objref.ObjectRef{Name: rule.Spec.Export.Service.Name, Namespace: rule.Spec.Export.Service.Namespace}
		mappings[exportRef] = append(mappings[exportRef], rule)
	}

	out = map[string][]objref.KMetadata{}
	svcs := d.hubInterface.ListServices(d.exportHubName)

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

	for _, svc := range svcs {
		origin := objref.KObj(svc)
		for _, rule := range mappings[origin] {
			destination := objref.ObjectRef{Name: rule.Spec.Import.Service.Name, Namespace: rule.Spec.Import.Service.Namespace}

			peerPortMapping := map[int32]int32{}

			for _, port := range svc.Spec.Ports {

				peerPort, err := d.hubInterface.GetPortPeer(d.importHubName, d.exportHubName, origin.Namespace, origin.Name, port.Port)
				if err != nil {
					return nil, err
				}
				peerPortMapping[port.Port] = peerPort

				tunnelName := fmt.Sprintf("%s-tunnel-%d-%d", rule.Name, port.Port, peerPort)
				hubsBound, err := d.hubsChain.Build(tunnelName, origin, destination, port.Port, peerPort, ways)
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

				allowName := fmt.Sprintf("%s-allows-%d-%d", rule.Name, port.Port, peerPort)
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

			serviceName := fmt.Sprintf("%s-service", rule.Name)

			ports := buildPorts(peerPortMapping, &svc.Spec)

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
	}

	for name := range out {
		out[name] = diffobjs.Unique(out[name])
	}
	return out, nil
}

func buildPorts(peerPortMapping map[int32]int32, spec *corev1.ServiceSpec) []discovery.MappingPort {
	ports := []discovery.MappingPort{}
	for _, port := range spec.Ports {
		if port.Protocol != corev1.ProtocolTCP {
			continue
		}
		svcPort := peerPortMapping[port.Port]
		ports = append(ports, discovery.MappingPort{
			Name:       port.Name,
			Port:       port.Port,
			Protocol:   string(port.Protocol),
			TargetPort: svcPort,
		})
	}
	return ports
}
