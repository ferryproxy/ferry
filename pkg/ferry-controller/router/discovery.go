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
	"strconv"
	"strings"

	"github.com/ferryproxy/ferry/pkg/consts"
	"github.com/ferryproxy/ferry/pkg/ferry-controller/router/resource"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// BuildServiceDiscovery the Egress Discovery resource, perhaps Service or DNS
func BuildServiceDiscovery(name, namespace string, labels map[string]string, peerPortMapping map[int32]int32, spec *corev1.ServiceSpec) ([]resource.Resourcer, error) {
	ports := []string{}
	resources := []resource.Resourcer{}

	meta := metav1.ObjectMeta{
		Name:      name,
		Namespace: namespace,
		Labels:    labels,
	}
	service := &corev1.Service{
		ObjectMeta: meta,
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{},
		},
	}

	for _, port := range spec.Ports {
		if port.Protocol != corev1.ProtocolTCP {
			continue
		}

		svcPort := peerPortMapping[port.Port]
		ports = append(ports, strconv.Itoa(int(svcPort)))

		service.Spec.Ports = append(service.Spec.Ports, corev1.ServicePort{
			Name:       port.Name,
			Port:       port.Port,
			Protocol:   port.Protocol,
			TargetPort: intstr.FromInt(int(svcPort)),
		})
	}

	labels[consts.LabelFerryExportedFromPortsKey] = strings.Join(ports, "-")

	resources = append(resources, resource.Service{service})
	return resources, nil
}
