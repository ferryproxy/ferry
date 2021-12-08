package controller

import (
	"context"
	"sync"
	"time"

	"github.com/ferry-proxy/ferry/pkg/router"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
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
	}
}

type DataPlaneController struct {
	mut                        sync.RWMutex
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
}

func (c *DataPlaneController) Run(ctx context.Context) {
	c.ctx = ctx
	cli := c.exportClientset.CoreV1().Services("")
	informer := cache.NewSharedInformer(&cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			options.LabelSelector = c.labelSelector
			return cli.List(ctx, options)
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			options.LabelSelector = c.labelSelector
			return cli.Watch(ctx, options)
		},
	}, &corev1.Service{}, 0*time.Second)
	informer.AddEventHandler(c)
	informer.Run(ctx.Done())
}

func (c *DataPlaneController) apply(ctx context.Context, svcs []*corev1.Service) error {
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

	err = ir.Apply(ctx, c.exportClientset)
	if err != nil {
		c.logger.Error(err, "Apply Server")
		return err
	}

	err = er.Apply(ctx, c.importClientset)
	if err != nil {
		c.logger.Error(err, "Apply Client")
		return err
	}
	return nil
}

func (c *DataPlaneController) delete(ctx context.Context, svcs []*corev1.Service) error {
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

	err = ir.Delete(ctx, c.exportClientset)
	if err != nil {
		c.logger.Error(err, "Delete Server")
		return err
	}

	err = er.Delete(ctx, c.importClientset)
	if err != nil {
		c.logger.Error(err, "Delete Client")
		return err
	}
	return nil
}

func (c *DataPlaneController) OnAdd(obj interface{}) {
	svc := obj.(*corev1.Service)
	c.logger.Info("OnAdd",
		"Service", svc.Name+"."+svc.Namespace,
		"ImportCluster", c.importClusterName,
		"ExportCluster", c.exportClusterName,
	)
	svcs := []*corev1.Service{svc}
	err := c.apply(c.ctx, svcs)
	if err != nil {
		c.logger.Error(err, "OnAdd",
			"Service", svc.Name+"."+svc.Namespace,
			"ImportCluster", c.importClusterName,
			"ExportCluster", c.exportClusterName,
		)
	}
}

func (c *DataPlaneController) OnUpdate(oldObj, newObj interface{}) {
	svc := newObj.(*corev1.Service)
	c.logger.Info("OnUpdate",
		"Service", svc.Name+"."+svc.Namespace,
		"ImportCluster", c.importClusterName,
		"ExportCluster", c.exportClusterName,
	)
	svcs := []*corev1.Service{svc}
	err := c.apply(c.ctx, svcs)
	if err != nil {
		c.logger.Error(err, "OnUpdate",
			"Service", svc.Name+"."+svc.Namespace,
			"ImportCluster", c.importClusterName,
			"ExportCluster", c.exportClusterName,
		)
	}

}

func (c *DataPlaneController) OnDelete(obj interface{}) {
	svc := obj.(*corev1.Service)
	c.logger.Info("OnDelete",
		"Service", svc.Name+"."+svc.Namespace,
		"ImportCluster", c.importClusterName,
		"ExportCluster", c.exportClusterName,
	)
	svcs := []*corev1.Service{svc}
	err := c.delete(c.ctx, svcs)
	if err != nil {
		c.logger.Error(err, "OnDelete",
			"Service", svc.Name+"."+svc.Namespace,
			"ImportCluster", c.importClusterName,
			"ExportCluster", c.exportClusterName,
		)
	}
}
