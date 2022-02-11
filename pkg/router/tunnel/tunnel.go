package tunnel

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/ferry-proxy/ferry/pkg/router"
	"github.com/ferry-proxy/ferry/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
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
func (serviceEgressDiscoveryBuilder) Build(proxy *router.Proxy, origin, destination utils.ObjectRef, spec *corev1.ServiceSpec) ([]router.Resourcer, error) {
	resources := []router.Resourcer{}
	addresses := router.BuildIPToEndpointAddress(proxy.InClusterEgressIPs)

	meta := metav1.ObjectMeta{
		Name:      destination.Name,
		Namespace: destination.Namespace,
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

	for _, port := range spec.Ports {
		if port.Protocol != corev1.ProtocolTCP {
			continue
		}
		svcPort := getPort(proxy.ImportClusterName, origin.Namespace, origin.Name, port.Port)
		if proxy.Reverse {
			svcPort += proxy.ExportPortOffset
		} else {
			svcPort += proxy.ImportPortOffset
		}
		portName := fmt.Sprintf("%s-%s-%d-%d", origin.Name, origin.Namespace, port.Port, svcPort)
		service.Spec.Ports = append(service.Spec.Ports, corev1.ServicePort{
			Name:     portName,
			Port:     port.Port,
			Protocol: port.Protocol,
			TargetPort: intstr.IntOrString{
				Type:   intstr.Int,
				IntVal: port.Port,
			},
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
	return resources, nil
}

var EgressBuilder = egressBuilder{}

type egressBuilder struct{}

func (egressBuilder) Build(proxy *router.Proxy, origin, destination utils.ObjectRef, spec *corev1.ServiceSpec) ([]router.Resourcer, error) {
	if proxy.Reverse {
		var serverBuilder serverBuilder
		return serverBuilder.Build(proxy, origin, destination, spec)
	} else {
		var clientBuilder clientBuilder
		return clientBuilder.Build(proxy, origin, destination, spec)
	}
}

type clientBuilder struct{}

// Build the client Egress resource
func (clientBuilder) Build(proxy *router.Proxy, origin, destination utils.ObjectRef, spec *corev1.ServiceSpec) ([]router.Resourcer, error) {
	labels := utils.MergeMap(proxy.Labels, map[string]string{
		"ferry-tunnel": "true",
	})

	resourcers := []router.Resourcer{}

	name := fmt.Sprintf("%s-%s-%s-%s-%s-%s-tunnel-client", proxy.ImportClusterName, destination.Namespace, destination.Name, proxy.ExportClusterName, origin.Namespace, origin.Name)
	configMap := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: proxy.TunnelNamespace,
			Labels:    labels,
		},
		Data: map[string]string{},
	}
	for _, port := range spec.Ports {
		if port.Protocol != corev1.ProtocolTCP {
			continue
		}
		svcPort := getPort(proxy.ImportClusterName, origin.Namespace, origin.Name, port.Port)
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
				origin.Name + "." + origin.Namespace + ".svc:" + strconv.FormatInt(int64(port.Port), 10),
			},
		}
		if proxy.Reverse {
			bind := "ssh://" + proxy.ImportIngressIPs[0] + ":" + strconv.FormatInt(int64(proxy.ImportIngressPort), 10)
			chain.Bind = append(chain.Bind, bind)
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

	return resourcers, nil
}

var IngressBuilder = ingressBuilder{}

type ingressBuilder struct{}

func (ingressBuilder) Build(proxy *router.Proxy, origin, destination utils.ObjectRef, spec *corev1.ServiceSpec) ([]router.Resourcer, error) {
	if proxy.Reverse {
		var clientBuilder clientBuilder
		return clientBuilder.Build(proxy, origin, destination, spec)
	} else {
		var serverBuilder serverBuilder
		return serverBuilder.Build(proxy, origin, destination, spec)
	}
}

type serverBuilder struct{}

// Build the server Deployment resource
func (serverBuilder) Build(proxy *router.Proxy, origin, destination utils.ObjectRef, spec *corev1.ServiceSpec) ([]router.Resourcer, error) {
	labels := utils.MergeMap(proxy.Labels, map[string]string{
		"ferry-tunnel": "true",
	})

	resourcers := []router.Resourcer{}

	name := fmt.Sprintf("%s-%s-%s-%s-%s-%s-tunnel-server", proxy.ImportClusterName, destination.Namespace, destination.Name, proxy.ExportClusterName, origin.Namespace, origin.Name)

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

	return resourcers, nil
}
