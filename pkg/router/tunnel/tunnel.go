package tunnel

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/ferry-proxy/ferry/pkg/router"
	"github.com/ferry-proxy/ferry/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var clusterPort = map[string]uint16{}
var cache = map[string]uint16{}

// TODO Calculate port based on cluster, namespace, and name
func getPort(cluster, namespace, name string, port int32) int32 {
	k := fmt.Sprintf("%s-%s-%d", namespace, name, port)
	if p, ok := cache[k]; ok {
		return int32(p)
	}
	i := clusterPort[cluster]
	i = i + 1
	clusterPort[cluster] = i
	cache[k] = i
	return int32(i)
}

type Chain struct {
	Bind  []string `json:"bind"`
	Proxy []string `json:"proxy"`
}

var ServiceEgressDiscoveryBuilder = serviceEgressDiscoveryBuilder{}

type serviceEgressDiscoveryBuilder struct {
}

// Build the Egress Discovery resource, perhaps Service or DNS
func (serviceEgressDiscoveryBuilder) Build(proxy *router.Proxy, destinationServices []*corev1.Service) ([]router.Resourcer, error) {
	resources := []router.Resourcer{}
	addresses := router.BuildIPToEndpointAddress(proxy.InClusterEgressIPs)
	for _, svc := range destinationServices {
		meta := metav1.ObjectMeta{
			Name:      svc.Name,
			Namespace: svc.Namespace,
			Labels:    proxy.Labels,
		}
		service := &corev1.Service{
			ObjectMeta: meta,
			Spec: corev1.ServiceSpec{
				Ports: []corev1.ServicePort{},
			},
		}
		endpoints := &corev1.Endpoints{
			ObjectMeta: meta,
			Subsets:    []corev1.EndpointSubset{},
		}

		for _, port := range svc.Spec.Ports {
			if port.Protocol != corev1.ProtocolTCP {
				continue
			}
			svcPort := getPort(proxy.ImportClusterName, svc.Namespace, svc.Name, port.Port)
			if proxy.Reverse {
				svcPort += proxy.ExportPortOffset
			} else {
				svcPort += proxy.ImportPortOffset
			}
			portName := fmt.Sprintf("%s-%s-%d-%d", svc.Name, svc.Namespace, port.Port, svcPort)
			service.Spec.Ports = append(service.Spec.Ports, corev1.ServicePort{
				Name:     portName,
				Port:     port.Port,
				Protocol: port.Protocol,
			})
			endpoints.Subsets = append(endpoints.Subsets, corev1.EndpointSubset{
				Addresses: addresses,
				Ports: []corev1.EndpointPort{
					{
						Name:     portName,
						Port:     svcPort,
						Protocol: port.Protocol,
					},
				},
			})
		}

		resources = append(resources, router.Service{service})
		resources = append(resources, router.Endpoints{endpoints})
	}
	return resources, nil
}

var EgressBuilder = egressBuilder{}

type egressBuilder struct{}

func (egressBuilder) Build(proxy *router.Proxy, destinationServices []*corev1.Service) ([]router.Resourcer, error) {
	if proxy.Reverse {
		var serverBuilder serverBuilder
		return serverBuilder.Build(proxy, destinationServices)
	} else {
		var clientBuilder clientBuilder
		return clientBuilder.Build(proxy, destinationServices)
	}
}

type clientBuilder struct{}

// Build the client Egress resource
func (clientBuilder) Build(proxy *router.Proxy, destinationServices []*corev1.Service) ([]router.Resourcer, error) {
	labels := utils.MergeMap(proxy.Labels, map[string]string{
		"ferry-tunnel": "true",
	})

	resourcers := []router.Resourcer{}

	for _, svc := range destinationServices {
		name := fmt.Sprintf("%s-%s-%s-%s-tunnel-client", proxy.ImportClusterName, proxy.ExportClusterName, svc.Namespace, svc.Name)
		configMap := corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: proxy.TunnelNamespace,
				Labels:    labels,
			},
			Data: map[string]string{},
		}
		for _, port := range svc.Spec.Ports {
			if port.Protocol != corev1.ProtocolTCP {
				continue
			}
			svcPort := getPort(proxy.ImportClusterName, svc.Namespace, svc.Name, port.Port)
			if proxy.Reverse {
				svcPort += proxy.ExportPortOffset
			} else {
				svcPort += proxy.ImportPortOffset
			}
			chain := Chain{
				Bind: []string{
					"0.0.0.0:" + strconv.FormatInt(int64(svcPort), 10),
				},
				Proxy: []string{
					svc.Name + "." + svc.Namespace + ".svc" + ":" + strconv.FormatInt(int64(port.Port), 10),
				},
			}
			if proxy.Reverse {
				proxy := "ssh://" + proxy.ImportIngressIPs[0] + ":" + strconv.FormatInt(int64(proxy.ImportIngressPort), 10)
				chain.Bind = append(chain.Bind, proxy)
			} else {
				proxy := "ssh://" + proxy.ExportIngressIPs[0] + ":" + strconv.FormatInt(int64(proxy.ExportIngressPort), 10)
				chain.Proxy = append(chain.Proxy, proxy)
			}
			data, err := json.MarshalIndent([]Chain{chain}, "", "  ")
			if err != nil {
				return nil, err
			}
			configMap.Data[strconv.FormatInt(int64(port.Port), 10)] = string(data)
		}
		resourcers = append(resourcers, router.ConfigMap{&configMap})
	}

	return resourcers, nil
}

var IngressBuilder = ingressBuilder{}

type ingressBuilder struct{}

func (ingressBuilder) Build(proxy *router.Proxy, destinationServices []*corev1.Service) ([]router.Resourcer, error) {
	if proxy.Reverse {
		var clientBuilder clientBuilder
		return clientBuilder.Build(proxy, destinationServices)
	} else {
		var serverBuilder serverBuilder
		return serverBuilder.Build(proxy, destinationServices)
	}
}

type serverBuilder struct{}

// Build the server Deployment resource
func (serverBuilder) Build(proxy *router.Proxy, destinationServices []*corev1.Service) ([]router.Resourcer, error) {
	labels := utils.MergeMap(proxy.Labels, map[string]string{
		"ferry-tunnel": "true",
	})

	resourcers := []router.Resourcer{}

	for _, svc := range destinationServices {
		name := fmt.Sprintf("%s-%s-%s-%s-tunnel-server", proxy.ImportClusterName, proxy.ExportClusterName, svc.Namespace, svc.Name)

		configMap := corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: proxy.TunnelNamespace,
				Labels:    labels,
			},
			Data: map[string]string{
				"tunnel-server": `
[
	{
		"bind": [
			"ssh://0.0.0.0:31087"
		],
		"proxy": [
			"-"
		]
	}
]
`,
			},
		}
		resourcers = append(resourcers, router.ConfigMap{&configMap})
	}

	return resourcers, nil
}
