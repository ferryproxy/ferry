package mapping

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ferry-proxy/api/apis/traffic/v1alpha2"
	"github.com/ferry-proxy/ferry/pkg/consts"
	"github.com/ferry-proxy/ferry/pkg/ferry-controller/router/resource"
	"github.com/ferry-proxy/ferry/pkg/ferry-controller/utils"
	"github.com/ferry-proxy/ferry/pkg/utils/objref"
	"github.com/ferry-proxy/ferry/pkg/utils/trybuffer"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

type ClusterCache interface {
	ListServices(name string) []*corev1.Service
	GetHub(name string) *v1alpha2.Hub
	GetIdentity(name string) string
	Clientset(name string) kubernetes.Interface
	LoadPortPeer(importHubName string, list *corev1.ServiceList)
	GetPortPeer(importHubName string, cluster, namespace, name string, port int32) int32
	RegistryServiceCallback(exportHubName, importHubName string, cb func())
	UnregistryServiceCallback(exportHubName, importHubName string)
}

type MappingControllerConfig struct {
	Namespace                  string
	ExportHubName              string
	ImportHubName              string
	ClusterCache               ClusterCache
	ExportClientset            kubernetes.Interface
	ImportClientset            kubernetes.Interface
	Logger                     logr.Logger
	SourceResourceBuilder      resource.ResourceBuilders
	DestinationResourceBuilder resource.ResourceBuilders
}

func NewMappingController(conf MappingControllerConfig) *MappingController {
	return &MappingController{
		namespace:                  conf.Namespace,
		importHubName:              conf.ImportHubName,
		exportHubName:              conf.ExportHubName,
		exportClientset:            conf.ExportClientset,
		importClientset:            conf.ImportClientset,
		logger:                     conf.Logger,
		clusterCache:               conf.ClusterCache,
		sourceResourceBuilder:      conf.SourceResourceBuilder,
		destinationResourceBuilder: conf.DestinationResourceBuilder,
		mappings:                   map[objref.ObjectRef][]objref.ObjectRef{},
	}
}

type MappingController struct {
	mut sync.Mutex
	ctx context.Context

	namespace string
	labels    map[string]string

	exportHubName string
	importHubName string

	mappings map[objref.ObjectRef][]objref.ObjectRef

	clusterCache ClusterCache

	exportClientset            kubernetes.Interface
	importClientset            kubernetes.Interface
	sourceResourceBuilder      resource.ResourceBuilders
	destinationResourceBuilder resource.ResourceBuilders
	lastSourceResources        []resource.Resourcer
	lastDestinationResources   []resource.Resourcer
	logger                     logr.Logger

	try *trybuffer.TryBuffer

	isClose bool
}

func (d *MappingController) Start(ctx context.Context) error {
	d.logger.Info("DataPlane controller started")
	defer func() {
		d.logger.Info("DataPlane controller stopped")
	}()
	d.ctx = ctx

	// Mark managed by ferry
	opt := metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(d.getLabel()).String(),
	}

	err := d.initLastSourceResources(ctx, opt)
	if err != nil {
		return err
	}
	err = d.initLastDestinationResources(ctx, opt)
	if err != nil {
		return err
	}

	d.try = trybuffer.NewTryBuffer(d.sync, time.Second/2)

	d.clusterCache.RegistryServiceCallback(d.exportHubName, d.importHubName, d.Sync)

	return nil
}

func (d *MappingController) Sync() {
	d.try.Try()
}

func (d *MappingController) SetRoutes(rules []*v1alpha2.Route) {
	mappings := map[objref.ObjectRef][]objref.ObjectRef{}
	for _, rule := range rules {
		exportRef := objref.ObjectRef{Name: rule.Spec.Export.Service.Name, Namespace: rule.Spec.Export.Service.Namespace}
		importRef := objref.ObjectRef{Name: rule.Spec.Import.Service.Name, Namespace: rule.Spec.Import.Service.Namespace}
		for _, v := range mappings[exportRef] {
			if v == importRef {
				return
			}
		}
		mappings[exportRef] = append(mappings[exportRef], importRef)
	}

	d.mut.Lock()
	defer d.mut.Unlock()
	d.mappings = mappings
}

func (d *MappingController) initLastSourceResources(ctx context.Context, opt metav1.ListOptions) error {
	cmList, err := d.exportClientset.
		CoreV1().
		ConfigMaps(d.namespace).
		List(ctx, opt)
	if err != nil {
		return err
	}
	for _, item := range cmList.Items {
		d.lastSourceResources = append(d.lastSourceResources, resource.ConfigMap{item.DeepCopy()})
	}
	return nil
}

func (d *MappingController) initLastDestinationResources(ctx context.Context, opt metav1.ListOptions) error {
	cmList, err := d.importClientset.
		CoreV1().
		ConfigMaps(d.namespace).
		List(ctx, opt)
	if err != nil {
		return err
	}
	for _, item := range cmList.Items {
		d.lastDestinationResources = append(d.lastDestinationResources, resource.ConfigMap{item.DeepCopy()})
	}
	svcList, err := d.importClientset.
		CoreV1().
		Services("").
		List(ctx, opt)
	if err != nil {
		return err
	}
	for _, item := range svcList.Items {
		d.lastDestinationResources = append(d.lastDestinationResources, resource.Service{item.DeepCopy()})
	}

	d.clusterCache.LoadPortPeer(d.importHubName, svcList)
	return nil
}

func (d *MappingController) getLabel() map[string]string {
	if d.labels != nil {
		return d.labels
	}
	d.labels = map[string]string{
		consts.LabelFerryManagedByKey:    consts.LabelFerryManagedByValue,
		consts.LabelFerryExportedFromKey: d.exportHubName,
		consts.LabelFerryImportedToKey:   d.importHubName,
	}
	return d.labels
}

func (d *MappingController) getProxyInfo() (*resource.Proxy, error) {
	exportHubName := d.exportHubName
	importHubName := d.importHubName

	proxy := &resource.Proxy{
		Labels:          d.getLabel(),
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
			gateway = mergeGateway(gateway, gw)
		}
	}

	if !gateway.Reachable {
		proxy.Reverse = true

		gatewayReverse := importCluster.Spec.Gateway
		if exportCluster.Spec.Override != nil {
			gw, ok := exportCluster.Spec.Override[exportHubName]
			if ok {
				gatewayReverse = mergeGateway(gatewayReverse, gw)
			}
		}

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
		proxy.ExportIngressAddress = gateway.Address
		proxy.ExportIdentity = d.clusterCache.GetIdentity(exportHubName)

		exportProxy, err := clusterProxies(d.clusterCache, gateway.Navigation)
		if err != nil {
			return nil, err
		}

		importProxy, err := clusterProxies(d.clusterCache, gateway.Reception)
		if err != nil {
			return nil, err
		}

		proxy.ExportProxy = exportProxy
		proxy.ImportProxy = importProxy
	}

	proxy.GetPortFunc = func(namespace, name string, port int32) int32 {
		return d.clusterCache.GetPortPeer(importHubName, exportCluster.Name, namespace, name, port)
	}

	return proxy, nil
}

func mergeGateway(origin, override v1alpha2.HubSpecGateway) v1alpha2.HubSpecGateway {
	origin.Reachable = override.Reachable
	if override.Address != "" {
		origin.Address = override.Address
	}
	if override.Navigation != nil {
		origin.Navigation = override.Navigation
	}
	if override.Reception != nil {
		origin.Reception = override.Reception
	}
	return origin
}

func (d *MappingController) sync() {
	d.mut.Lock()
	defer d.mut.Unlock()

	if d.isClose {
		return
	}
	ctx := d.ctx

	if len(d.lastSourceResources) == 0 && len(d.lastDestinationResources) == 0 &&
		len(d.mappings) == 0 {
		d.logger.Info("No need to sync")
		return
	}

	var ir []resource.Resourcer
	var er []resource.Resourcer

	proxy, err := d.getProxyInfo()
	if err != nil {
		d.logger.Error(err, "get proxy info failed")
		return
	}

	svcs := d.clusterCache.ListServices(d.exportHubName)

	d.logger.Info("Sync", "ServicesCount", len(svcs))

	for _, svc := range svcs {
		origin := objref.KObj(svc)

		for _, destination := range d.mappings[origin] {
			i, err := d.sourceResourceBuilder.Build(proxy, origin, destination, &svc.Spec)
			if err != nil {
				d.logger.Error(err, "sourceResourceBuilder", "origin", origin, "destination", destination)
			}
			ir = append(ir, i...)

			e, err := d.destinationResourceBuilder.Build(proxy, origin, destination, &svc.Spec)
			if err != nil {
				d.logger.Error(err, "destinationResourceBuilder", "origin", origin, "destination", destination)
			}
			er = append(er, e...)
		}
	}

	d.logger.Info("CalculatePatchResources",
		"lastSourceResources", len(d.lastSourceResources),
		"lastDestinationResources", len(d.lastDestinationResources),
		"ImportResources", len(ir),
		"ExportResources", len(er),
	)

	if len(d.lastSourceResources) == 0 && len(d.lastDestinationResources) == 0 &&
		len(ir) == 0 && len(er) == 0 {
		d.logger.Info("No need to sync")
		return
	}

	sourceUpdate, sourceDelete := utils.CalculatePatchResources(d.lastSourceResources, ir)
	destinationUpdate, destinationDelete := utils.CalculatePatchResources(d.lastDestinationResources, er)

	if len(sourceUpdate) == 0 && len(sourceDelete) == 0 && len(destinationUpdate) == 0 && len(destinationDelete) == 0 {
		d.logger.Info("No need to sync")
		return
	}

	d.logger.Info("Sync",
		"SourceUpdate", len(sourceUpdate),
		"SourceDelete", len(sourceDelete),
		"DestinationUpdate", len(destinationUpdate),
		"DestinationDelete", len(destinationDelete),
	)

	for _, r := range sourceUpdate {
		err := r.Apply(ctx, d.exportClientset)
		if err != nil {
			d.logger.Error(err, "Apply Export")
		}
	}

	for _, r := range destinationUpdate {
		err := r.Apply(ctx, d.importClientset)
		if err != nil {
			d.logger.Error(err, "Apply Import")
		}
	}

	for _, r := range sourceDelete {
		err := r.Delete(ctx, d.exportClientset)
		if err != nil {
			d.logger.Error(err, "Delete Export")
		}
	}

	for _, r := range destinationDelete {
		err := r.Delete(ctx, d.importClientset)
		if err != nil {
			d.logger.Error(err, "Delete Import")
		}
	}

	d.lastSourceResources = ir
	d.lastDestinationResources = er
	return
}

func (d *MappingController) Close() {
	d.mut.Lock()
	defer d.mut.Unlock()

	if d.isClose {
		return
	}
	d.isClose = true
	d.clusterCache.UnregistryServiceCallback(d.exportHubName, d.importHubName)
	d.try.Close()

	ctx := context.Background()

	for _, r := range d.lastSourceResources {
		err := r.Delete(ctx, d.exportClientset)
		if err != nil {
			d.logger.Error(err, "Delete Export")
		}
	}

	for _, r := range d.lastDestinationResources {
		err := r.Delete(ctx, d.importClientset)
		if err != nil {
			d.logger.Error(err, "Delete Import")
		}
	}
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
