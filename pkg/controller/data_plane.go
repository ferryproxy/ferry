package controller

import (
	"context"
	"sync"
	"time"

	"github.com/ferry-proxy/ferry/pkg/router"
	"github.com/ferry-proxy/ferry/pkg/utils"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
)

type DataPlaneControllerConfig struct {
	ImportClusterName          string
	ExportClusterName          string
	Selector                   labels.Selector
	ExportClientset            *kubernetes.Clientset
	ImportClientset            *kubernetes.Clientset
	Logger                     logr.Logger
	Proxy                      router.Proxy
	SourceResourceBuilder      router.ResourceBuilders
	DestinationResourceBuilder router.ResourceBuilders
}

func NewDataPlaneController(conf DataPlaneControllerConfig) *DataPlaneController {
	return &DataPlaneController{
		importClusterName:          conf.ImportClusterName,
		exportClusterName:          conf.ExportClusterName,
		exportClientset:            conf.ExportClientset,
		importClientset:            conf.ImportClientset,
		logger:                     conf.Logger,
		labelSelector:              conf.Selector.String(),
		proxy:                      conf.Proxy,
		sourceResourceBuilder:      conf.SourceResourceBuilder,
		destinationResourceBuilder: conf.DestinationResourceBuilder,
		cache:                      map[string]*corev1.Service{},
		chSync:                     make(chan struct{}, 1),
	}
}

type DataPlaneController struct {
	mut                        sync.Mutex
	ctx                        context.Context
	importClusterName          string
	exportClusterName          string
	logger                     logr.Logger
	labelSelector              string
	exportClientset            *kubernetes.Clientset
	importClientset            *kubernetes.Clientset
	proxy                      router.Proxy
	sourceResourceBuilder      router.ResourceBuilders
	destinationResourceBuilder router.ResourceBuilders
	cache                      map[string]*corev1.Service

	lastSourceResources      []router.Resourcer
	lastDestinationResources []router.Resourcer
	chSync                   chan struct{}
}

func (d *DataPlaneController) Start(ctx context.Context) error {
	d.logger.Info("DataPlane controller started")
	defer func() {
		d.logger.Info("DataPlane controller stopped")
	}()
	d.ctx = ctx
	informerFactory := informers.NewSharedInformerFactoryWithOptions(d.exportClientset, 0,
		informers.WithTweakListOptions(func(options *metav1.ListOptions) {
			options.LabelSelector = d.labelSelector
		}))
	informer := informerFactory.Core().V1().Services().Informer()
	informer.AddEventHandler(d)

	opt := metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(d.proxy.Labels).String(),
	}

	err := d.initLastSourceResources(ctx, opt)
	if err != nil {
		return err
	}
	err = d.initLastDestinationResources(ctx, opt)
	if err != nil {
		return err
	}

	err = d.initCache(ctx, metav1.ListOptions{
		LabelSelector: d.labelSelector,
	})
	if err != nil {
		return err
	}

	go informer.Run(ctx.Done())
	go func() {
		for {
			select {
			case <-d.chSync:
			next:
				for {
					select {
					case <-d.chSync:
					case <-time.After(time.Second):
						break next
					case <-ctx.Done():
						return
					}
				}
			case <-ctx.Done():
				return
			}
			err := d.sync(ctx)
			if err != nil {
				d.logger.Error(err, "Sync failed")
			}
		}
	}()
	d.trySync()
	return nil
}

func (d *DataPlaneController) initCache(ctx context.Context, opt metav1.ListOptions) error {
	svcList, err := d.exportClientset.CoreV1().Services("").List(ctx, opt)
	if err != nil {
		return err
	}
	for _, item := range svcList.Items {
		svc := item.DeepCopy()
		d.cache[uniqueKey(svc.Name, svc.Namespace)] = svc
	}
	return nil
}

func (d *DataPlaneController) initLastSourceResources(ctx context.Context, opt metav1.ListOptions) error {
	cmList, err := d.exportClientset.CoreV1().ConfigMaps(d.proxy.TunnelNamespace).List(ctx, opt)
	if err != nil {
		return err
	}
	for _, item := range cmList.Items {
		d.lastSourceResources = append(d.lastSourceResources, router.ConfigMap{item.DeepCopy()})
	}
	return nil
}

func (d *DataPlaneController) initLastDestinationResources(ctx context.Context, opt metav1.ListOptions) error {
	cmList, err := d.importClientset.CoreV1().ConfigMaps(d.proxy.TunnelNamespace).List(ctx, opt)
	if err != nil {
		return err
	}
	for _, item := range cmList.Items {
		d.lastDestinationResources = append(d.lastDestinationResources, router.ConfigMap{item.DeepCopy()})
	}
	svcList, err := d.importClientset.CoreV1().Services("").List(ctx, opt)
	if err != nil {
		return err
	}
	for _, item := range svcList.Items {
		d.lastDestinationResources = append(d.lastDestinationResources, router.Service{item.DeepCopy()})
	}
	epList, err := d.importClientset.CoreV1().Endpoints("").List(ctx, opt)
	if err != nil {
		return err
	}
	for _, item := range epList.Items {
		d.lastDestinationResources = append(d.lastDestinationResources, router.Endpoints{item.DeepCopy()})
	}
	return nil
}

func (d *DataPlaneController) trySync() {
	select {
	case d.chSync <- struct{}{}:
	default:
	}
}

func (d *DataPlaneController) sync(ctx context.Context) error {
	svcs := []*corev1.Service{}
	for _, svc := range d.cache {
		svcs = append(svcs, svc)
	}
	d.logger.Info("Sync", "ServicesCount", len(svcs))
	ir, err := d.sourceResourceBuilder.Build(&d.proxy, svcs)
	if err != nil {
		d.logger.Error(err, "Server Build")
		return err
	}

	er, err := d.destinationResourceBuilder.Build(&d.proxy, svcs)
	if err != nil {
		d.logger.Error(err, "Client Build")
		return err
	}

	sourceUpdate, sourceDelete := router.CalculatePatchResources(d.lastSourceResources, ir)
	destinationUpdate, destinationDelete := router.CalculatePatchResources(d.lastDestinationResources, er)

	for _, r := range sourceUpdate {
		err := r.Apply(ctx, d.exportClientset)
		if err != nil {
			d.logger.Error(err, "Apply Export")
			return err
		}
	}

	for _, r := range destinationUpdate {
		err := r.Apply(ctx, d.importClientset)
		if err != nil {
			d.logger.Error(err, "Apply Import")
			return err
		}
	}

	// TODO remove this
	time.Sleep(5 * time.Second)

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

func (d *DataPlaneController) OnAdd(obj interface{}) {
	svc := obj.(*corev1.Service)
	d.logger.Info("OnAdd",
		"Service", utils.KObj(svc),
	)
	svc = svc.DeepCopy()

	d.mut.Lock()
	defer d.mut.Unlock()
	d.cache[uniqueKey(svc.Name, svc.Namespace)] = svc
	d.trySync()

}

func (d *DataPlaneController) OnUpdate(oldObj, newObj interface{}) {
	svc := newObj.(*corev1.Service)
	d.logger.Info("OnUpdate",
		"Service", utils.KObj(svc),
	)
	svc = svc.DeepCopy()

	d.mut.Lock()
	defer d.mut.Unlock()
	d.cache[uniqueKey(svc.Name, svc.Namespace)] = svc
	d.trySync()
}

func (d *DataPlaneController) OnDelete(obj interface{}) {
	svc := obj.(*corev1.Service)
	d.logger.Info("OnDelete",
		"Service", utils.KObj(svc),
	)
	svc = svc.DeepCopy()

	d.mut.Lock()
	defer d.mut.Unlock()
	delete(d.cache, uniqueKey(svc.Name, svc.Namespace))
	d.trySync()
}

func (d *DataPlaneController) Cleanup(ctx context.Context) {
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
