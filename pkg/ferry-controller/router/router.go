package router

import (
	"fmt"

	"github.com/ferry-proxy/api/apis/traffic/v1alpha2"
	"github.com/ferry-proxy/ferry/pkg/consts"
	"github.com/ferry-proxy/ferry/pkg/ferry-controller/router/resource"
	"github.com/ferry-proxy/ferry/pkg/ferry-controller/router/tunnel"
	"github.com/ferry-proxy/ferry/pkg/utils/objref"
	corev1 "k8s.io/api/core/v1"
)

type ClusterCache interface {
	ListServices(name string) []*corev1.Service
	GetHub(name string) *v1alpha2.Hub
	GetIdentity(name string) string
	GetPortPeer(importHubName string, cluster, namespace, name string, port int32) int32
}

type RouterConfig struct {
	Namespace     string
	Labels        map[string]string
	ExportHubName string
	ImportHubName string
	ClusterCache  ClusterCache
}

func NewRouter(conf RouterConfig) *Router {
	return &Router{
		namespace:                  conf.Namespace,
		labels:                     conf.Labels,
		importHubName:              conf.ImportHubName,
		exportHubName:              conf.ExportHubName,
		clusterCache:               conf.ClusterCache,
		mappings:                   map[objref.ObjectRef][]objref.ObjectRef{},
		sourceResourceBuilder:      resource.ResourceBuilders{tunnel.IngressBuilder},
		destinationResourceBuilder: resource.ResourceBuilders{tunnel.EgressBuilder, tunnel.ServiceEgressDiscoveryBuilder},
	}
}

type Router struct {
	namespace string
	labels    map[string]string

	exportHubName string
	importHubName string

	mappings map[objref.ObjectRef][]objref.ObjectRef

	clusterCache ClusterCache

	sourceResourceBuilder      resource.ResourceBuilders
	destinationResourceBuilder resource.ResourceBuilders
}

func (d *Router) SetRoutes(rules []*v1alpha2.Route) {
	mappings := map[objref.ObjectRef][]objref.ObjectRef{}
	for _, rule := range rules {
		exportRef := objref.ObjectRef{Name: rule.Spec.Export.Service.Name, Namespace: rule.Spec.Export.Service.Namespace}
		importRef := objref.ObjectRef{Name: rule.Spec.Import.Service.Name, Namespace: rule.Spec.Import.Service.Namespace}
		mappings[exportRef] = append(mappings[exportRef], importRef)
	}
	d.mappings = mappings
}

func (d *Router) getProxyInfo() (*resource.Proxy, error) {
	exportHubName := d.exportHubName
	importHubName := d.importHubName

	proxy := &resource.Proxy{
		Labels:          d.labels,
		RemotePrefix:    consts.FerryName,
		TunnelNamespace: d.namespace,

		ExportHubName: exportHubName,
		ImportHubName: importHubName,
	}

	exportCluster := d.clusterCache.GetHub(exportHubName)
	gateway := exportCluster.Spec.Gateway

	importCluster := d.clusterCache.GetHub(importHubName)
	if importCluster.Spec.Override != nil {
		gw, ok := importCluster.Spec.Override[exportHubName]
		if ok {
			gateway = gw
		}
	}

	if gateway.Reachable {
		proxy.ExportIngressAddress = gateway.Address
		proxy.ExportIdentity = d.clusterCache.GetIdentity(exportHubName)

		importProxy, err := clusterProxies(d.clusterCache, gateway.Navigation)
		if err != nil {
			return nil, err
		}

		exportProxy, err := clusterProxies(d.clusterCache, gateway.Reception)
		if err != nil {
			return nil, err
		}

		proxy.ExportProxy = exportProxy
		proxy.ImportProxy = importProxy
	} else {
		proxy.Reverse = true

		gatewayReverse := importCluster.Spec.Gateway
		if exportCluster.Spec.Override != nil {
			gw, ok := exportCluster.Spec.Override[exportHubName]
			if ok {
				gatewayReverse = gw
			}
		}
		if gatewayReverse.Reachable {

			proxy.ImportIngressAddress = gatewayReverse.Address
			proxy.ImportIdentity = d.clusterCache.GetIdentity(importHubName)

			importProxy, err := clusterProxies(d.clusterCache, gatewayReverse.Navigation)
			if err != nil {
				return nil, err
			}

			exportProxy, err := clusterProxies(d.clusterCache, gatewayReverse.Reception)
			if err != nil {
				return nil, err
			}
			proxy.ExportProxy = exportProxy
			proxy.ImportProxy = importProxy
		} else {
			proxy.Repeater = true

			importProxy, err := clusterProxies(d.clusterCache, gateway.Reception)
			if err != nil {
				return nil, err
			}

			exportProxy, err := clusterProxies(d.clusterCache, gatewayReverse.Navigation)
			if err != nil {
				return nil, err
			}

			proxy.ExportProxy = exportProxy
			proxy.ImportProxy = importProxy
		}
	}

	proxy.GetPortFunc = func(namespace, name string, port int32) int32 {
		return d.clusterCache.GetPortPeer(importHubName, exportCluster.Name, namespace, name, port)
	}

	return proxy, nil
}

func (d *Router) BuildResource() (ingressResource, egressResource []resource.Resourcer, err error) {

	proxy, err := d.getProxyInfo()
	if err != nil {
		return nil, nil, fmt.Errorf("get proxy info failed: %w", err)
	}

	svcs := d.clusterCache.ListServices(d.exportHubName)

	for _, svc := range svcs {
		origin := objref.KObj(svc)

		for _, destination := range d.mappings[origin] {
			i, err := d.sourceResourceBuilder.Build(proxy, origin, destination, &svc.Spec)
			if err != nil {
				return nil, nil, err
			}
			ingressResource = append(ingressResource, i...)

			e, err := d.destinationResourceBuilder.Build(proxy, origin, destination, &svc.Spec)
			if err != nil {
				return nil, nil, err
			}
			egressResource = append(egressResource, e...)
		}
	}

	return
}

func clusterProxy(clusterCache ClusterCache, proxy v1alpha2.HubSpecGatewayWay) (string, error) {
	if proxy.Proxy != "" {
		return proxy.Proxy, nil
	}

	ci := clusterCache.GetHub(proxy.HubName)
	if ci == nil {
		return "", fmt.Errorf("failed get cluster information %q", proxy.HubName)
	}
	if ci.Spec.Gateway.Address == "" {
		return "", fmt.Errorf("failed get address of cluster information %q", proxy.HubName)
	}
	address := ci.Spec.Gateway.Address
	return "ssh://" + address + "?identity_data=" + clusterCache.GetIdentity(proxy.HubName), nil
}

func clusterProxies(clusterCache ClusterCache, proxies v1alpha2.HubSpecGatewayWays) ([]string, error) {
	out := make([]string, 0, len(proxies))
	for _, proxy := range proxies {
		p, err := clusterProxy(clusterCache, proxy)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}
