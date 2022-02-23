package controller

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/ferry-proxy/api/apis/ferry/v1alpha1"
	"github.com/ferry-proxy/ferry/pkg/consts"
	"github.com/ferry-proxy/ferry/pkg/router"
	"github.com/ferry-proxy/ferry/pkg/utils"
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
		mappings:                     map[utils.ObjectRef][]utils.ObjectRef{},
		labels:                       map[string]labels.Selector{},
	}
}

type DataPlaneController struct {
	mut sync.Mutex
	ctx context.Context

	exportClusterName string
	importClusterName string

	exportCluster *v1alpha1.ClusterInformation
	importCluster *v1alpha1.ClusterInformation

	mappings map[utils.ObjectRef][]utils.ObjectRef
	labels   map[string]labels.Selector

	clusterInformationController *clusterInformationController

	exportClientset            *kubernetes.Clientset
	importClientset            *kubernetes.Clientset
	sourceResourceBuilder      router.ResourceBuilders
	destinationResourceBuilder router.ResourceBuilders
	lastSourceResources        []router.Resourcer
	lastDestinationResources   []router.Resourcer
	logger                     logr.Logger

	try *utils.TryBuffer

	isClose bool
}

func (d *DataPlaneController) Start(ctx context.Context) error {
	d.logger.Info("DataPlane controller started")
	defer func() {
		d.logger.Info("DataPlane controller stopped")
	}()
	d.ctx = ctx

	proxy, err := d.getProxyInfo(ctx)
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

	d.try = utils.NewTryBuffer(func() {
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

func (d *DataPlaneController) RegistrySelector(sel labels.Selector) {
	d.mut.Lock()
	defer d.mut.Unlock()
	d.labels[sel.String()] = sel
}

func (d *DataPlaneController) UnregistrySelector(sel labels.Selector) {
	d.mut.Lock()
	defer d.mut.Unlock()
	delete(d.labels, sel.String())
}

func (d *DataPlaneController) RegistryObj(export, impor utils.ObjectRef) {
	d.mut.Lock()
	defer d.mut.Unlock()

	for _, v := range d.mappings[export] {
		if v == impor {
			return
		}
	}
	d.mappings[export] = append(d.mappings[export], impor)
}

func (d *DataPlaneController) UnregistryObj(export, impor utils.ObjectRef) {
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

	epList, err := d.importClientset.
		CoreV1().
		Endpoints("").
		List(ctx, opt)
	if err != nil {
		return err
	}
	for _, item := range epList.Items {
		d.lastDestinationResources = append(d.lastDestinationResources, router.Endpoints{item.DeepCopy()})
	}
	return nil
}

func CalculateProxy(exportProxy, importProxy []string) ([]string, []string) {
	if len(exportProxy) == 0 && len(importProxy) == 0 {
		return nil, nil
	}
	if len(exportProxy) == 0 {
		return []string{importProxy[0]}, importProxy
	}
	if len(importProxy) == 0 {
		return exportProxy, []string{exportProxy[0]}
	}
	if exportProxy[0] == importProxy[0] {
		return exportProxy, importProxy
	}
	return exportProxy, append([]string{exportProxy[0]}, importProxy...)
}

func (d *DataPlaneController) getProxyInfo(ctx context.Context) (*router.Proxy, error) {
	exportClusterName := d.exportClusterName
	importClusterName := d.importClusterName

	exportClientset := d.clusterInformationController.Clientset(exportClusterName)
	if exportClientset == nil {
		return nil, fmt.Errorf("not found clientset %q", exportClusterName)
	}
	importClientset := d.clusterInformationController.Clientset(importClusterName)
	if importClientset == nil {
		return nil, fmt.Errorf("not found clientset %q", importClusterName)
	}

	exportCluster := d.clusterInformationController.Get(exportClusterName)
	if exportCluster == nil {
		return nil, fmt.Errorf("not found cluster information %q", exportCluster)
	}

	importCluster := d.clusterInformationController.Get(importClusterName)
	if importCluster == nil {
		return nil, fmt.Errorf("not found cluster information %q", importClusterName)
	}

	inClusterEgressIPs, err := getIPs(ctx, importClientset, importCluster.Spec.Egress)
	if err != nil {
		return nil, err
	}

	exportIngressIPs, err := getIPs(ctx, exportClientset, exportCluster.Spec.Ingress)
	if err != nil {
		return nil, err
	}

	exportIngressPort, err := getPort(ctx, exportClientset, exportCluster.Spec.Ingress)
	if err != nil {
		return nil, err
	}

	importIngressIPs, err := getIPs(ctx, importClientset, importCluster.Spec.Ingress)
	if err != nil {
		return nil, err
	}
	importIngressPort, err := getPort(ctx, importClientset, importCluster.Spec.Ingress)
	if err != nil {
		return nil, err
	}

	reverse := false

	var exportProxy = []string{}
	var importProxy = []string{}

	var exportProxies v1alpha1.Proxies
	var importProxies v1alpha1.Proxies
	var ok bool

	if exportCluster.Spec.Ingress != nil {
		if len(exportCluster.Spec.Ingress.Proxies) != 0 {
			exportProxies, ok = exportCluster.Spec.Ingress.Proxies[importClusterName]
			if !ok {
				exportProxies = exportCluster.Spec.Ingress.DefaultProxies
			}
		} else {
			exportProxies = exportCluster.Spec.Ingress.DefaultProxies
		}
	}

	if importCluster.Spec.Egress != nil {
		if len(importCluster.Spec.Egress.Proxies) != 0 {
			importProxies, ok = importCluster.Spec.Egress.Proxies[exportClusterName]
			if !ok {
				importProxies = importCluster.Spec.Egress.DefaultProxies
			}
		} else {
			importProxies = importCluster.Spec.Egress.DefaultProxies
		}
	}

	if len(exportIngressIPs) == 0 {
		if len(importIngressIPs) == 0 {
			if len(importProxies) == 0 && len(exportProxies) == 0 {
				return nil, fmt.Errorf("not found ingress ip or proxy")
			} else {
				exportProxy, err = d.clusterInformationController.proxies(ctx, exportProxies)
				if err != nil {
					return nil, err
				}
				importProxy, err = d.clusterInformationController.proxies(ctx, importProxies)
				if err != nil {
					return nil, err
				}
				exportProxy, importProxy = CalculateProxy(exportProxy, importProxy)
			}
		} else {
			reverse = true
		}
	}

	ports := d.clusterInformationController.TunnelPorts(importClusterName)
	return &router.Proxy{
		Labels: map[string]string{
			consts.LabelFerryManagedByKey:    consts.LabelFerryManagedByValue,
			consts.LabelFerryExportedFromKey: exportCluster.Name,
			consts.LabelFerryImportedToKey:   importCluster.Name,
		},
		RemotePrefix:    "ferry",
		TunnelNamespace: "ferry-tunnel-system",
		Reverse:         reverse,

		ExportClusterName: exportCluster.Name,
		ImportClusterName: importCluster.Name,

		InClusterEgressIPs: inClusterEgressIPs,

		ExportIngressIPs:  exportIngressIPs,
		ExportIngressPort: exportIngressPort,

		ImportIngressIPs:  importIngressIPs,
		ImportIngressPort: importIngressPort,

		ExportProxy: exportProxy,
		ImportProxy: importProxy,
		GetPortFunc: func(namespace, name string, port int32) int32 {
			return ports.getPort(exportCluster.Name, namespace, name, port)
		},
	}, nil
}

func (d *DataPlaneController) sync(ctx context.Context) error {
	d.mut.Lock()
	defer d.mut.Unlock()

	if d.isClose {
		return nil
	}

	if len(d.lastSourceResources) == 0 && len(d.lastDestinationResources) == 0 &&
		len(d.labels) == 0 && len(d.mappings) == 0 {
		d.logger.Info("No need to sync")
		return nil
	}

	var ir []router.Resourcer
	var er []router.Resourcer

	proxy, err := d.getProxyInfo(ctx)
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
		origin := utils.KObj(svc)

		for _, label := range d.labels {
			if label.Matches(labels.Set(svc.Labels)) {
				i, err := d.sourceResourceBuilder.Build(proxy, origin, origin, &svc.Spec)
				if err != nil {
					d.logger.Error(err, "sourceResourceBuilder", "origin", origin, "destination", origin)
					return err
				}
				ir = append(ir, i...)

				e, err := d.destinationResourceBuilder.Build(proxy, origin, origin, &svc.Spec)
				if err != nil {
					d.logger.Error(err, "destinationResourceBuilder", "origin", origin, "destination", origin)
				}
				er = append(er, e...)
				break
			}
		}

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

	sourceUpdate, sourceDelete := router.CalculatePatchResources(d.lastSourceResources, ir)
	destinationUpdate, destinationDelete := router.CalculatePatchResources(d.lastDestinationResources, er)

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
		ServiceCache(d.importClusterName).
		UnregistryOnAdd(d.importClusterName)

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
