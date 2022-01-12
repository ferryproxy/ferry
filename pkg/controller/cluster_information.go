package controller

import (
	"context"
	"sync"

	"github.com/ferry-proxy/api/apis/ferry/v1alpha1"
	versioned "github.com/ferry-proxy/client-go/generated/clientset/versioned"
	externalversions "github.com/ferry-proxy/client-go/generated/informers/externalversions"
	"github.com/ferry-proxy/ferry/pkg/utils"
	"github.com/go-logr/logr"
	restclient "k8s.io/client-go/rest"
)

type clusterInformationControllerConfig struct {
	Logger    logr.Logger
	Config    *restclient.Config
	Namespace string
	SyncFunc  func(context.Context, string)
}
type clusterInformationController struct {
	mut       sync.RWMutex
	ctx       context.Context
	logger    logr.Logger
	config    *restclient.Config
	cache     map[string]*v1alpha1.ClusterInformation
	syncFunc  func(context.Context, string)
	namespace string
}

func newClusterInformationController(conf *clusterInformationControllerConfig) *clusterInformationController {
	return &clusterInformationController{
		config:    conf.Config,
		namespace: conf.Namespace,
		logger:    conf.Logger,
		syncFunc:  conf.SyncFunc,
		cache:     map[string]*v1alpha1.ClusterInformation{},
	}
}

func (c *clusterInformationController) Run(ctx context.Context) error {
	c.logger.Info("ClusterInformation controller started")
	defer c.logger.Info("ClusterInformation controller stopped")

	clientset, err := versioned.NewForConfig(c.config)
	if err != nil {
		return err
	}
	c.ctx = ctx
	informerFactory := externalversions.NewSharedInformerFactoryWithOptions(clientset, 0,
		externalversions.WithNamespace(c.namespace))
	informer := informerFactory.Ferry().
		V1alpha1().
		ClusterInformations().
		Informer()
	informer.AddEventHandler(c)
	informer.Run(ctx.Done())
	return nil
}

func (c *clusterInformationController) OnAdd(obj interface{}) {
	f := obj.(*v1alpha1.ClusterInformation)
	f = f.DeepCopy()
	c.logger.Info("OnAdd",
		"ClusterInformation", utils.KObj(f),
	)

	c.mut.Lock()
	defer c.mut.Unlock()

	c.cache[f.Name] = f
	c.syncFunc(c.ctx, f.Name)
}

func (c *clusterInformationController) OnUpdate(oldObj, newObj interface{}) {
	f := newObj.(*v1alpha1.ClusterInformation)
	f = f.DeepCopy()
	c.logger.Info("OnUpdate",
		"ClusterInformation", utils.KObj(f),
	)

	c.mut.Lock()
	defer c.mut.Unlock()

	c.cache[f.Name] = f
	c.syncFunc(c.ctx, f.Name)
}

func (c *clusterInformationController) OnDelete(obj interface{}) {
	f := obj.(*v1alpha1.ClusterInformation)
	c.logger.Info("OnDelete",
		"ClusterInformation", utils.KObj(f),
	)

	c.mut.Lock()
	defer c.mut.Unlock()

	delete(c.cache, f.Name)
	c.syncFunc(c.ctx, f.Name)
}

func (c *clusterInformationController) Get(name string) *v1alpha1.ClusterInformation {
	c.mut.RLock()
	defer c.mut.RUnlock()
	return c.cache[name]
}
