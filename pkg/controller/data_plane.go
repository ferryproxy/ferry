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
}

func (c *DataPlaneController) Start(ctx context.Context) error {
	c.logger.Info("DataPlane controller started")
	defer func() {
		c.logger.Info("DataPlane controller stopped")
	}()
	c.ctx = ctx
	informerFactory := informers.NewSharedInformerFactoryWithOptions(c.exportClientset, 0,
		informers.WithTweakListOptions(func(options *metav1.ListOptions) {
			options.LabelSelector = c.labelSelector
		}))
	informer := informerFactory.Core().V1().Services().Informer()
	informer.AddEventHandler(c)

	opt := metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(c.proxy.Labels).String(),
	}

	err := c.initLastSourceResources(ctx, opt)
	if err != nil {
		return err
	}
	err = c.initLastDestinationResources(ctx, opt)
	if err != nil {
		return err
	}

	err = c.initCache(ctx, metav1.ListOptions{
		LabelSelector: c.labelSelector,
	})
	if err != nil {
		return err
	}

	go informer.Run(ctx.Done())
	return nil
}

func (c *DataPlaneController) initCache(ctx context.Context, opt metav1.ListOptions) error {
	svcList, err := c.exportClientset.CoreV1().Services("").List(ctx, opt)
	if err != nil {
		return err
	}
	for _, item := range svcList.Items {
		svc := item.DeepCopy()
		c.cache[uniqueKey(svc.Name, svc.Namespace)] = svc
	}
	return nil
}

func (c *DataPlaneController) initLastSourceResources(ctx context.Context, opt metav1.ListOptions) error {
	cmList, err := c.exportClientset.CoreV1().ConfigMaps(c.proxy.TunnelNamespace).List(ctx, opt)
	if err != nil {
		return err
	}
	for _, item := range cmList.Items {
		c.lastSourceResources = append(c.lastSourceResources, router.ConfigMap{item.DeepCopy()})
	}
	return nil
}

func (c *DataPlaneController) initLastDestinationResources(ctx context.Context, opt metav1.ListOptions) error {
	cmList, err := c.importClientset.CoreV1().ConfigMaps(c.proxy.TunnelNamespace).List(ctx, opt)
	if err != nil {
		return err
	}
	for _, item := range cmList.Items {
		c.lastDestinationResources = append(c.lastDestinationResources, router.ConfigMap{item.DeepCopy()})
	}
	svcList, err := c.importClientset.CoreV1().Services("").List(ctx, opt)
	if err != nil {
		return err
	}
	for _, item := range svcList.Items {
		c.lastDestinationResources = append(c.lastDestinationResources, router.Service{item.DeepCopy()})
	}
	epList, err := c.importClientset.CoreV1().Endpoints("").List(ctx, opt)
	if err != nil {
		return err
	}
	for _, item := range epList.Items {
		c.lastDestinationResources = append(c.lastDestinationResources, router.Endpoints{item.DeepCopy()})
	}
	return nil
}

func (c *DataPlaneController) Sync(ctx context.Context) error {
	c.logger.Info("Sync")
	svcs := []*corev1.Service{}
	for _, svc := range c.cache {
		svcs = append(svcs, svc)
	}
	ir, err := c.sourceResourceBuilder.Build(&c.proxy, svcs)
	if err != nil {
		c.logger.Error(err, "Server Build")
		return err
	}

	er, err := c.destinationResourceBuilder.Build(&c.proxy, svcs)
	if err != nil {
		c.logger.Error(err, "Client Build")
		return err
	}

	sourceUpdate, sourceDelete := router.CalculatePatchResources(c.lastSourceResources, ir)
	destinationUpdate, destinationDelete := router.CalculatePatchResources(c.lastDestinationResources, er)

	for _, r := range sourceUpdate {
		err := r.Apply(ctx, c.exportClientset)
		if err != nil {
			c.logger.Error(err, "Apply Export")
			return err
		}
	}

	for _, r := range destinationUpdate {
		err := r.Apply(ctx, c.importClientset)
		if err != nil {
			c.logger.Error(err, "Apply Import")
			return err
		}
	}

	// TODO remove this
	time.Sleep(5 * time.Second)

	for _, r := range sourceDelete {
		err := r.Delete(ctx, c.exportClientset)
		if err != nil {
			c.logger.Error(err, "Delete Export")
		}
	}

	for _, r := range destinationDelete {
		err := r.Delete(ctx, c.importClientset)
		if err != nil {
			c.logger.Error(err, "Delete Import")
		}
	}

	c.lastSourceResources = ir
	c.lastDestinationResources = er
	return nil
}

func (c *DataPlaneController) OnAdd(obj interface{}) {
	svc := obj.(*corev1.Service)
	c.logger.Info("OnAdd",
		"Service", utils.KObj(svc),
	)
	svc = svc.DeepCopy()

	c.mut.Lock()
	defer c.mut.Unlock()
	c.cache[uniqueKey(svc.Name, svc.Namespace)] = svc
	err := c.Sync(c.ctx)
	if err != nil {
		c.logger.Error(err, "OnAdd",
			"Service", utils.KObj(svc),
		)
	}
}

func (c *DataPlaneController) OnUpdate(oldObj, newObj interface{}) {
	svc := newObj.(*corev1.Service)
	c.logger.Info("OnUpdate",
		"Service", utils.KObj(svc),
	)
	svc = svc.DeepCopy()

	c.mut.Lock()
	defer c.mut.Unlock()
	c.cache[uniqueKey(svc.Name, svc.Namespace)] = svc
	err := c.Sync(c.ctx)
	if err != nil {
		c.logger.Error(err, "OnUpdate",
			"Service", utils.KObj(svc),
		)
	}
}

func (c *DataPlaneController) OnDelete(obj interface{}) {
	svc := obj.(*corev1.Service)
	c.logger.Info("OnDelete",
		"Service", utils.KObj(svc),
	)
	svc = svc.DeepCopy()

	c.mut.Lock()
	defer c.mut.Unlock()
	delete(c.cache, uniqueKey(svc.Name, svc.Namespace))
	err := c.Sync(c.ctx)
	if err != nil {
		c.logger.Error(err, "OnDelete",
			"Service", utils.KObj(svc),
		)
	}
}
