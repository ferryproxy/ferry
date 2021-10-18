package original

import (
	"github.com/DaoCloud-OpenSource/ferry/pkg/router"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ServiceEgressDiscoveryBuilder struct{}

// Build the Egress Discovery resource, perhaps Service or DNS
func (ServiceEgressDiscoveryBuilder) Build(proxy *router.Proxy, destinationServices []*corev1.Service) (router.Resourcer, error) {
	backends := router.Backends{}
	for _, svc := range destinationServices {
		for _, port := range svc.Spec.Ports {
			if port.Protocol != corev1.ProtocolTCP {
				continue
			}
			backend := router.BuildBackend(proxy, svc.Name, svc.Namespace, proxy.EgressIPs, port.Port, proxy.EgressPort)
			backends = append(backends, backend)
		}
	}
	return backends, nil
}

type EgressBuilder struct{}

// Build the client Egress resource
func (EgressBuilder) Build(proxy *router.Proxy, destinationServices []*corev1.Service) (router.Resourcer, error) {
	backends := router.Backends{}
	ingresses := router.Ingresses{}
	for _, svc := range destinationServices {
		for _, port := range svc.Spec.Ports {
			if port.Protocol != corev1.ProtocolTCP {
				continue
			}
			domain := svc.Name + "." + svc.Namespace
			rules := []networkingv1.IngressRule{
				{
					Host: domain,
				},
				{
					Host: domain + ".svc",
				},
			}

			egressName := proxy.RemotePrefix + "-egress-" + svc.Name
			ingress := &networkingv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name:      egressName,
					Namespace: svc.Namespace,
					Labels:    proxy.Labels,
				},
				Spec: networkingv1.IngressSpec{
					DefaultBackend: &networkingv1.IngressBackend{
						Service: &networkingv1.IngressServiceBackend{
							Name: egressName,
							Port: networkingv1.ServiceBackendPort{
								Number: port.Port,
							},
						},
					},
					Rules: rules,
				},
			}
			ingresses = append(ingresses, router.Ingress{Ingress: ingress})
			backend := router.BuildBackend(proxy, egressName, svc.Namespace, proxy.IngressIPs, port.Port, proxy.IngressPort)
			backends = append(backends, backend)
		}
	}
	return router.Resourcers{ingresses, backends}, nil
}

type IngressBuilder struct{}

// Build the server Ingress resource
func (IngressBuilder) Build(proxy *router.Proxy, destinationServices []*corev1.Service) (router.Resourcer, error) {
	ingresses := router.Ingresses{}
	for _, svc := range destinationServices {
		for _, port := range svc.Spec.Ports {
			if port.Protocol != corev1.ProtocolTCP {
				continue
			}
			domain := svc.Name + "." + svc.Namespace
			rules := []networkingv1.IngressRule{
				{
					Host: domain,
				},
				{
					Host: domain + ".svc",
				},
			}

			igressName := proxy.RemotePrefix + "-igress-" + svc.Name
			ingress := &networkingv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name:      igressName,
					Namespace: svc.Namespace,
					Labels:    proxy.Labels,
				},
				Spec: networkingv1.IngressSpec{
					DefaultBackend: &networkingv1.IngressBackend{
						Service: &networkingv1.IngressServiceBackend{
							Name: svc.Name,
							Port: networkingv1.ServiceBackendPort{
								Number: port.Port,
							},
						},
					},
					Rules: rules,
				},
			}
			ingresses = append(ingresses, router.Ingress{Ingress: ingress})

		}
	}
	return ingresses, nil
}
