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
