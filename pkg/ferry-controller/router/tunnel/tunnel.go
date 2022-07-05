package tunnel

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/ferry-proxy/ferry/pkg/consts"
	"github.com/ferry-proxy/ferry/pkg/ferry-controller/router"
	"github.com/ferry-proxy/ferry/pkg/utils/maps"
	"github.com/ferry-proxy/ferry/pkg/utils/objref"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type Chain struct {
	Bind  []string `json:"bind"`
	Proxy []string `json:"proxy"`
}

var ServiceEgressDiscoveryBuilder = serviceEgressDiscoveryBuilder{}

type serviceEgressDiscoveryBuilder struct {
}

// Build the Egress Discovery resource, perhaps Service or DNS
func (serviceEgressDiscoveryBuilder) Build(proxy *router.Proxy, origin, destination objref.ObjectRef, spec *corev1.ServiceSpec) ([]router.Resourcer, error) {
	labels := maps.Merge(proxy.Labels, map[string]string{
		consts.LabelFerryExportedFromNamespaceKey: origin.Namespace,
		consts.LabelFerryExportedFromNameKey:      origin.Name,
		consts.LabelFerryTunnelKey:                consts.LabelFerryTunnelValue,
	})

	ports := []string{}
	resources := []router.Resourcer{}

	meta := metav1.ObjectMeta{
		Name:      destination.Name,
		Namespace: destination.Namespace,
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

		svcPort := proxy.GetPortFunc(origin.Namespace, origin.Name, port.Port)
		ports = append(ports, strconv.Itoa(int(svcPort)))

		portName := fmt.Sprintf("%s-%s-%d-%d", origin.Name, origin.Namespace, port.Port, svcPort)
		service.Spec.Ports = append(service.Spec.Ports, corev1.ServicePort{
			Name:       portName,
			Port:       port.Port,
			Protocol:   port.Protocol,
			TargetPort: intstr.FromInt(int(svcPort)),
		})
	}
	sort.Strings(ports)
	labels[consts.LabelFerryExportedFromPortsKey] = strings.Join(ports, "-")

	resources = append(resources, router.Service{service})
	return resources, nil
}

var EgressBuilder = egressBuilder{}

type egressBuilder struct{}

func (egressBuilder) Build(proxy *router.Proxy, origin, destination objref.ObjectRef, spec *corev1.ServiceSpec) ([]router.Resourcer, error) {
	if len(proxy.ImportProxy) != 0 {
		var clientProxyBuilder clientProxyBuilder
		return clientProxyBuilder.Build(proxy, origin, destination, spec)
	}

	if proxy.Reverse {
		var serverBuilder serverBuilder
		return serverBuilder.Build(proxy, origin, destination, spec)
	} else {
		var clientBuilder clientBuilder
		return clientBuilder.Build(proxy, origin, destination, spec)
	}
}

type clientProxyBuilder struct{}

// Build the client proxy resource
func (clientProxyBuilder) Build(proxy *router.Proxy, origin, destination objref.ObjectRef, spec *corev1.ServiceSpec) ([]router.Resourcer, error) {
	labels := maps.Merge(proxy.Labels, map[string]string{
		consts.LabelFerryExportedFromNamespaceKey: origin.Namespace,
		consts.LabelFerryExportedFromNameKey:      origin.Name,
		consts.LabelFerryTunnelKey:                consts.LabelFerryTunnelValue,
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
		svcPort := proxy.GetPortFunc(origin.Namespace, origin.Name, port.Port)
		virtualName := fmt.Sprintf("unix:///dev/shm/%s-%s-%s-%s-%s-%s-%d-%d-tunnel.socks", proxy.ImportClusterName, destination.Namespace, destination.Name, proxy.ExportClusterName, origin.Namespace, origin.Name, port.Port, svcPort)

		chain := Chain{
			Bind: []string{
				"0.0.0.0:" + strconv.FormatInt(int64(svcPort), 10),
			},
			Proxy: append([]string{virtualName}, proxy.ImportProxy...),
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

type clientBuilder struct{}

// Build the client resource
func (clientBuilder) Build(proxy *router.Proxy, origin, destination objref.ObjectRef, spec *corev1.ServiceSpec) ([]router.Resourcer, error) {
	labels := maps.Merge(proxy.Labels, map[string]string{
		consts.LabelFerryExportedFromNamespaceKey: origin.Namespace,
		consts.LabelFerryExportedFromNameKey:      origin.Name,
		consts.LabelFerryTunnelKey:                consts.LabelFerryTunnelValue,
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
		svcPort := proxy.GetPortFunc(origin.Namespace, origin.Name, port.Port)
		chain := Chain{
			Bind: []string{
				"0.0.0.0:" + strconv.FormatInt(int64(svcPort), 10),
			},
			Proxy: []string{
				origin.Name + "." + origin.Namespace + ".svc:" + strconv.FormatInt(int64(port.Port), 10),
			},
		}
		if proxy.Reverse {
			bind := "ssh://" + proxy.ImportIngressAddress + "?identity_data=" + proxy.ImportIdentity
			chain.Bind = append(chain.Bind, bind)
		} else {
			proxy := "ssh://" + proxy.ExportIngressAddress + "?identity_data=" + proxy.ExportIdentity
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

func (ingressBuilder) Build(proxy *router.Proxy, origin, destination objref.ObjectRef, spec *corev1.ServiceSpec) ([]router.Resourcer, error) {
	if len(proxy.ExportProxy) != 0 {
		var serverProxyBuilder serverProxyBuilder
		return serverProxyBuilder.Build(proxy, origin, destination, spec)
	}

	if proxy.Reverse {
		var clientBuilder clientBuilder
		return clientBuilder.Build(proxy, origin, destination, spec)
	} else {
		var serverBuilder serverBuilder
		return serverBuilder.Build(proxy, origin, destination, spec)
	}
}

type serverProxyBuilder struct{}

// Build the server proxy resource
func (serverProxyBuilder) Build(proxy *router.Proxy, origin, destination objref.ObjectRef, spec *corev1.ServiceSpec) ([]router.Resourcer, error) {
	labels := maps.Merge(proxy.Labels, map[string]string{
		consts.LabelFerryExportedFromNamespaceKey: origin.Namespace,
		consts.LabelFerryExportedFromNameKey:      origin.Name,
		consts.LabelFerryTunnelKey:                consts.LabelFerryTunnelValue,
	})

	resourcers := []router.Resourcer{}

	name := fmt.Sprintf("%s-%s-%s-%s-%s-%s-tunnel-server", proxy.ImportClusterName, destination.Namespace, destination.Name, proxy.ExportClusterName, origin.Namespace, origin.Name)

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

		svcPort := proxy.GetPortFunc(origin.Namespace, origin.Name, port.Port)
		virtualName := fmt.Sprintf("unix:///dev/shm/%s-%s-%s-%s-%s-%s-%d-%d-tunnel.socks", proxy.ImportClusterName, destination.Namespace, destination.Name, proxy.ExportClusterName, origin.Namespace, origin.Name, port.Port, svcPort)

		chain := Chain{
			Bind: append([]string{virtualName}, proxy.ExportProxy...),
			Proxy: []string{
				origin.Name + "." + origin.Namespace + ".svc:" + strconv.FormatInt(int64(port.Port), 10),
			},
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

type serverBuilder struct{}

// Build the server resource
func (serverBuilder) Build(proxy *router.Proxy, origin, destination objref.ObjectRef, spec *corev1.ServiceSpec) ([]router.Resourcer, error) {
	labels := maps.Merge(proxy.Labels, map[string]string{
		consts.LabelFerryExportedFromNamespaceKey: origin.Namespace,
		consts.LabelFerryExportedFromNameKey:      origin.Name,
		consts.LabelFerryTunnelKey:                consts.LabelFerryTunnelValue,
	})

	resourcers := []router.Resourcer{}

	name := fmt.Sprintf("%s-%s-%s-%s-%s-%s-tunnel-server", proxy.ImportClusterName, destination.Namespace, destination.Name, proxy.ExportClusterName, origin.Namespace, origin.Name)

	configMap := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: proxy.TunnelNamespace,
			Labels:    labels,
		},
	}

	resourcers = append(resourcers, router.ConfigMap{&configMap})

	return resourcers, nil
}