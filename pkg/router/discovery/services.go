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

package discovery

import (
	"encoding/json"

	"github.com/ferryproxy/ferry/pkg/utils/objref"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BuildServiceDiscovery the Egress Discovery resource, perhaps Service or DNS
func BuildServiceDiscovery(om metav1.ObjectMeta, ips []string, mappingPorts map[string][]MappingPort) []objref.KMetadata {
	svc := corev1.Service{
		ObjectMeta: om,
	}
	ep := corev1.Endpoints{
		ObjectMeta: om,
	}

	type pair struct {
		Name     string
		Protocol string
	}
	uniq := map[pair]struct{}{}
	addresses := buildIPToEndpointAddress(ips)
	for _, ports := range mappingPorts {
		es := corev1.EndpointSubset{
			Addresses: addresses,
		}
		for _, port := range ports {
			es.Ports = append(es.Ports, corev1.EndpointPort{
				Name:     port.Name,
				Protocol: corev1.Protocol(port.Protocol),
				Port:     port.TargetPort,
			})

			key := pair{
				Name:     port.Name,
				Protocol: port.Protocol,
			}
			if _, ok := uniq[key]; ok {
				continue
			}
			uniq[key] = struct{}{}
			svc.Spec.Ports = append(svc.Spec.Ports, corev1.ServicePort{
				Name:     port.Name,
				Protocol: corev1.Protocol(port.Protocol),
				Port:     port.Port,
			})
		}

		if len(es.Ports) != 0 {
			ep.Subsets = append(ep.Subsets, es)
		}
	}

	return []objref.KMetadata{&svc, &ep}
}

func buildIPToEndpointAddress(ips []string) []corev1.EndpointAddress {
	eps := make([]corev1.EndpointAddress, 0, len(ips))
	for _, ip := range ips {
		eps = append(eps, corev1.EndpointAddress{
			IP: ip,
		})
	}
	return eps
}

type MappingPort struct {
	Name       string `json:"name,omitempty"`
	Protocol   string `json:"protocol,omitempty"`
	Port       int32  `json:"port,omitempty"`
	TargetPort int32  `json:"targetPort,omitempty"`
}

type Service struct {
	ExportHubName          string
	ExportServiceName      string
	ExportServiceNamespace string
	ImportServiceName      string
	ImportServiceNamespace string
	Ports                  []MappingPort
}

func ServiceFrom(m map[string]string) (Service, error) {
	s := Service{}
	err := json.Unmarshal([]byte(m["ports"]), &s.Ports)
	if err != nil {
		return s, err
	}
	s.ExportHubName = m["export_hub_name"]
	s.ExportServiceName = m["export_service_name"]
	s.ExportServiceNamespace = m["export_service_namespace"]
	s.ImportServiceName = m["import_service_name"]
	s.ImportServiceNamespace = m["import_service_namespace"]
	return s, nil
}
func (s Service) ToMap() (map[string]string, error) {
	portData, err := json.Marshal(s.Ports)
	if err != nil {
		return nil, err
	}
	out := map[string]string{
		"export_hub_name":          s.ExportHubName,
		"export_service_name":      s.ExportServiceName,
		"export_service_namespace": s.ExportServiceNamespace,
		"import_service_name":      s.ImportServiceName,
		"import_service_namespace": s.ImportServiceNamespace,
		"ports":                    string(portData),
	}
	return out, nil
}
