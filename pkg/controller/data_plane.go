package controller

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/ferry-proxy/api/apis/ferry/v1alpha1"
	"github.com/ferry-proxy/ferry/pkg/consts"
	"github.com/ferry-proxy/ferry/pkg/router"
	"github.com/ferry-proxy/ferry/pkg/utils"
	"github.com/ferry-proxy/utils/objref"
	"github.com/ferry-proxy/utils/trybuffer"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

type DataPlaneControllerConfig struct {
	ExportClusterName            string
	ImportClusterName            string
	ClusterInformationController *clusterInformationController
	ExportCluster                *v1alpha1.ClusterInformation
	ImportCluster                *v1alpha1.ClusterInformation
	ExportClientset              *kubernetes.Clientset
	ImportClientset              *kubernetes.Clientset
	Logger                       logr.Logger
	SourceResourceBuilder        router.ResourceBuilders
	DestinationResourceBuilder   router.ResourceBuilders
}

func NewDataPlaneController(conf DataPlaneControllerConfig) *DataPlaneController {
	return &DataPlaneController{
		importClusterName:            conf.ImportClusterName,
		exportClusterName:            conf.ExportClusterName,
		exportCluster:                conf.ExportCluster,
		importCluster:                conf.ImportCluster,
		exportClientset:              conf.ExportClientset,
		importClientset:              conf.ImportClientset,
		logger:                       conf.Logger,
		clusterInformationController: conf.ClusterInformationController,
		sourceResourceBuilder:        conf.SourceResourceBuilder,
		destinationResourceBuilder:   conf.DestinationResourceBuilder,
		mappings:                     map[objref.ObjectRef][]objref.ObjectRef{},
	}
}

type DataPlaneController struct {
	mut sync.Mutex
	ctx context.Context

	exportClusterName string
	importClusterName string

	exportCluster *v1alpha1.ClusterInformation
	importCluster *v1alpha1.ClusterInformation

	mappings map[objref.ObjectRef][]objref.ObjectRef

	clusterInformationController *clusterInformationController

	exportClientset            *kubernetes.Clientset
	importClientset            *kubernetes.Clientset
	sourceResourceBuilder      router.ResourceBuilders
	destinationResourceBuilder router.ResourceBuilders
	lastSourceResources        []router.Resourcer
	lastDestinationResources   []router.Resourcer
	logger                     logr.Logger

	try *trybuffer.TryBuffer

	isClose bool
}

func (d *DataPlaneController) Start(ctx context.Context) error {
	d.logger.Info("DataPlane controller started")
	defer func() {
		d.logger.Info("DataPlane controller stopped")
	}()
	d.ctx = ctx

	proxy, err := d.GetProxyInfo(ctx)
	if err != nil {
		return err
	}
	// Mark managed by ferry
	opt := metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(proxy.Labels).String(),
	}

	err = d.initLastSourceResources(ctx, proxy, opt)
	if err != nil {
		return err
	}
	err = d.initLastDestinationResources(ctx, proxy, opt)
	if err != nil {
		return err
	}

	d.try = trybuffer.NewTryBuffer(func() {
		err := d.sync(ctx)
		if err != nil {
			d.logger.Error(err, "sync failed")
		}
	}, time.Second/2)

	d.clusterInformationController.
		ServiceCache(d.exportClusterName).
		RegistryCallback(d.importClusterName, d.try.Try)

	return nil
}

func (d *DataPlaneController) Registry(export, impor objref.ObjectRef) {
	d.mut.Lock()
	defer d.mut.Unlock()

	for _, v := range d.mappings[export] {
		if v == impor {
			return
		}
	}
	d.mappings[export] = append(d.mappings[export], impor)
}

func (d *DataPlaneController) Unregistry(export, impor objref.ObjectRef) {
	d.mut.Lock()
	defer d.mut.Unlock()

	for i, v := range d.mappings[export] {
		if v == impor {
			d.mappings[export] = append(d.mappings[export][:i], d.mappings[export][i+1:]...)
			return
		}
	}
}

func (d *DataPlaneController) initLastSourceResources(ctx context.Context, proxy *router.Proxy, opt metav1.ListOptions) error {
	cmList, err := d.exportClientset.
		CoreV1().
		ConfigMaps(proxy.TunnelNamespace).
		List(ctx, opt)
	if err != nil {
		return err
	}
	for _, item := range cmList.Items {
		d.lastSourceResources = append(d.lastSourceResources, router.ConfigMap{item.DeepCopy()})
	}
	return nil
}

func (d *DataPlaneController) initLastDestinationResources(ctx context.Context, proxy *router.Proxy, opt metav1.ListOptions) error {
	cmList, err := d.importClientset.
		CoreV1().
		ConfigMaps(proxy.TunnelNamespace).
		List(ctx, opt)
	if err != nil {
		return err
	}
	for _, item := range cmList.Items {
		d.lastDestinationResources = append(d.lastDestinationResources, router.ConfigMap{item.DeepCopy()})
	}
	svcList, err := d.importClientset.
		CoreV1().
		Services("").
		List(ctx, opt)
	if err != nil {
		return err
	}
	for _, item := range svcList.Items {
		d.lastDestinationResources = append(d.lastDestinationResources, router.Service{item.DeepCopy()})
	}

	tunnelPorts := d.clusterInformationController.
		TunnelPorts(d.importClusterName)
	tunnelPorts.loadPortPeer(svcList)
	return nil
}

func (d *DataPlaneController) GetProxyInfo(ctx context.Context) (*router.Proxy, error) {
	proxy, err := d.getProxyInfo(ctx)
	if err != nil {
		for {
			d.logger.Error(err, "get proxy info failed")
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(2 * time.Second):
				proxy, err = d.getProxyInfo(ctx)
				if err != nil {
					continue
				}
				return proxy, nil
			}
		}
	}
	return proxy, nil
}

func (d *DataPlaneController) getProxyInfo(ctx context.Context) (*router.Proxy, error) {
	exportClusterName := d.exportClusterName
	importClusterName := d.importClusterName

	proxy := &router.Proxy{
		Labels: map[string]string{
			consts.LabelFerryManagedByKey:    consts.LabelFerryManagedByValue,
			consts.LabelFerryExportedFromKey: exportClusterName,
			consts.LabelFerryImportedToKey:   importClusterName,
		},
		RemotePrefix:    "ferry",
		TunnelNamespace: "ferry-tunnel-system",

		ExportClusterName: exportClusterName,
		ImportClusterName: importClusterName,
	}

	exportCluster := d.clusterInformationController.Get(exportClusterName)
	gateway := exportCluster.Spec.Gateway

	importCluster := d.clusterInformationController.Get(importClusterName)
	if importCluster.Spec.Override != nil {
		gw, ok := importCluster.Spec.Override[exportClusterName]
		if ok {
			gateway = mergeGateway(gateway, gw)
		}
	}

	if !gateway.Reachable {
		proxy.Reverse = true

		gatewayReverse := importCluster.Spec.Gateway
		if exportCluster.Spec.Override != nil {
			gw, ok := exportCluster.Spec.Override[exportClusterName]
			if ok {
				gatewayReverse = mergeGateway(gatewayReverse, gw)
			}
		}

		proxy.ImportIngressAddress = gatewayReverse.Address
		proxy.ImportIdentity = d.clusterInformationController.GetIdentity(importClusterName)

		importProxy, err := d.clusterInformationController.proxies(gatewayReverse.Navigation)
		if err != nil {
			return nil, err
		}

		exportProxy, err := d.clusterInformationController.proxies(gatewayReverse.Reception)
		if err != nil {
			return nil, err
		}
		proxy.ExportProxy = exportProxy
		proxy.ImportProxy = importProxy
	} else {
		proxy.ExportIngressAddress = gateway.Address
		proxy.ExportIdentity = d.clusterInformationController.GetIdentity(exportClusterName)

		exportProxy, err := d.clusterInformationController.proxies(gateway.Navigation)
		if err != nil {
			return nil, err
		}

		importProxy, err := d.clusterInformationController.proxies(gateway.Reception)
		if err != nil {
			return nil, err
		}

		proxy.ExportProxy = exportProxy
		proxy.ImportProxy = importProxy
	}

	ports := d.clusterInformationController.TunnelPorts(importClusterName)
	proxy.GetPortFunc = func(namespace, name string, port int32) int32 {
		return ports.getPort(exportCluster.Name, namespace, name, port)
	}

	return proxy, nil
}

func mergeGateway(origin, override v1alpha1.ClusterInformationSpecGateway) v1alpha1.ClusterInformationSpecGateway {
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

func (d *DataPlaneController) sync(ctx context.Context) error {
	d.mut.Lock()
	defer d.mut.Unlock()

	if d.isClose {
		return nil
	}

	if len(d.lastSourceResources) == 0 && len(d.lastDestinationResources) == 0 &&
		len(d.mappings) == 0 {
		d.logger.Info("No need to sync")
		return nil
	}

	var ir []router.Resourcer
	var er []router.Resourcer

	proxy, err := d.GetProxyInfo(ctx)
	if err != nil {
		return err
	}

	svcs := []*corev1.Service{}

	d.clusterInformationController.
		ServiceCache(d.exportClusterName).
		ForEach(func(svc *corev1.Service) {
			svcs = append(svcs, svc)
		})
	d.logger.Info("Sync", "ServicesCount", len(svcs))

	sort.Slice(svcs, func(i, j int) bool {
		return svcs[i].CreationTimestamp.Before(&svcs[j].CreationTimestamp)
	})

	for _, svc := range svcs {
		origin := objref.KObj(svc)

		for _, destination := range d.mappings[origin] {
			i, err := d.sourceResourceBuilder.Build(proxy, origin, destination, &svc.Spec)
			if err != nil {
				d.logger.Error(err, "sourceResourceBuilder", "origin", origin, "destination", destination)
				return err
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
		return nil
	}

	sourceUpdate, sourceDelete := utils.CalculatePatchResources(d.lastSourceResources, ir)
	destinationUpdate, destinationDelete := utils.CalculatePatchResources(d.lastDestinationResources, er)

	if len(sourceUpdate) == 0 && len(sourceDelete) == 0 && len(destinationUpdate) == 0 && len(destinationDelete) == 0 {
		d.logger.Info("No need to sync")
		return nil
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
	return nil
}

func (d *DataPlaneController) Close() {
	d.mut.Lock()
	defer d.mut.Unlock()
	d.isClose = true
	d.try.Close()

	d.clusterInformationController.
		ServiceCache(d.exportClusterName).
		UnregistryCallback(d.importClusterName)

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
