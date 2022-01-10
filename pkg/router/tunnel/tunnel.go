package tunnel

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/ferry-proxy/ferry/pkg/router"
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
func (serviceEgressDiscoveryBuilder) Build(proxy *router.Proxy, destinationServices []*corev1.Service) (router.Resourcer, error) {
	backends := router.Backends{}
	for _, svc := range destinationServices {
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
			backend := BuildBackend(proxy, svc.Name, svc.Namespace, proxy.InClusterEgressIPs, port.Port, svcPort)
			backends = append(backends, backend)
		}
	}
	return backends, nil
}

func BuildBackend(proxy *router.Proxy, name, ns string, ips []string, srcPort, destPort int32) router.Backend {
	meta := metav1.ObjectMeta{
		Name:      name,
		Namespace: ns,
		Labels:    proxy.Labels,
	}

	return router.Backend{
		Service: &corev1.Service{
			ObjectMeta: meta,
			Spec: corev1.ServiceSpec{
				Ports: []corev1.ServicePort{
					{
						Port:       srcPort,
						TargetPort: intstr.FromInt(int(destPort)),
					},
				},
			},
		},
		Endpoints: &corev1.Endpoints{
			ObjectMeta: meta,
			Subsets: []corev1.EndpointSubset{
				{
					Addresses: router.BuildIPToEndpointAddress(ips),
					Ports: []corev1.EndpointPort{
						{
							Port: destPort,
						},
					},
				},
			},
		},
	}
}

var EgressBuilder = egressBuilder{}

type egressBuilder struct{}

func (egressBuilder) Build(proxy *router.Proxy, destinationServices []*corev1.Service) (router.Resourcer, error) {
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
func (clientBuilder) Build(proxy *router.Proxy, destinationServices []*corev1.Service) (router.Resourcer, error) {
	configMap := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      proxy.ImportClusterName + "-" + proxy.ExportClusterName + "-client-tunnel",
			Namespace: "ferry-tunnel-system",
			Labels: map[string]string{
				"ferry-tunnel": "true",
			},
		},
		Data: map[string]string{},
	}

	for _, svc := range destinationServices {
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
			name := strings.Replace(fmt.Sprintf("%s-%s-%d", svc.Namespace, svc.Name, port.Port), ".", "-", -1)
			configMap.Data[name] = string(data)
		}
	}

	return router.Resourcers{
		router.ConfigMap{&configMap},
	}, nil
}

var IngressBuilder = ingressBuilder{}

type ingressBuilder struct{}

func (ingressBuilder) Build(proxy *router.Proxy, destinationServices []*corev1.Service) (router.Resourcer, error) {
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
func (serverBuilder) Build(proxy *router.Proxy, destinationServices []*corev1.Service) (router.Resourcer, error) {
	configMap := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      proxy.ImportClusterName + "-" + proxy.ExportClusterName + "-server-tunnel",
			Namespace: "ferry-tunnel-system",
			Labels: map[string]string{
				"ferry-tunnel": "true",
			},
		},
		Data: map[string]string{
			"ingress": `
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

	return router.Resourcers{
		router.ConfigMap{&configMap},
	}, nil
}
