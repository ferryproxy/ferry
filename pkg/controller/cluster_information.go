package controller

import (
	"context"
	"sync"

	"github.com/ferry-proxy/ferry/api/v1alpha1"
	"github.com/go-logr/logr"
	restclient "k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/cache"
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
	cache, err := cache.New(c.config, cache.Options{
		Namespace: c.namespace,
	})
	if err != nil {
		return err
	}
	informer, err := cache.GetInformer(ctx, &v1alpha1.ClusterInformation{})
	if err != nil {
		return err
	}
	informer.AddEventHandler(c)
	c.ctx = ctx
	return cache.Start(ctx)
}

func (c *clusterInformationController) OnAdd(obj interface{}) {
	f := obj.(*v1alpha1.ClusterInformation)
	f = f.DeepCopy()
	c.logger.Info("OnAdd",
		"ClusterInformation", uniqueKey(f.Name, f.Namespace),
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
		"ClusterInformation", uniqueKey(f.Name, f.Namespace),
	)

	c.mut.Lock()
	defer c.mut.Unlock()

	c.cache[f.Name] = f
	c.syncFunc(c.ctx, f.Name)
}

func (c *clusterInformationController) OnDelete(obj interface{}) {
	f := obj.(*v1alpha1.ClusterInformation)
	c.logger.Info("OnDelete",
		"ClusterInformation", uniqueKey(f.Name, f.Namespace),
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
