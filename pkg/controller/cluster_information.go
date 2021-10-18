package controller

import (
	"context"
	"sync"

	"github.com/go-logr/logr"

	"github.com/DaoCloud-OpenSource/ferry/api/v1alpha1"
	restclient "k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/cache"
)

type clusterInformationControllerConfig struct {
	Logger    logr.Logger
	Config    *restclient.Config
	Namespace string
}
type clusterInformationController struct {
	mut       sync.RWMutex
	ctx       context.Context
	logger    logr.Logger
	config    *restclient.Config
	mapping   map[string]*v1alpha1.ClusterInformation
	namespace string
}

func newClusterInformationController(conf *clusterInformationControllerConfig) *clusterInformationController {
	return &clusterInformationController{
		config:    conf.Config,
		namespace: conf.Namespace,
		logger:    conf.Logger,
		mapping:   map[string]*v1alpha1.ClusterInformation{},
	}
}

func (c *clusterInformationController) Run(ctx context.Context) error {
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
	c.mut.Lock()
	defer c.mut.Unlock()

	f := obj.(*v1alpha1.ClusterInformation)
	f = f.DeepCopy()
	c.logger.Info("OnAdd",
		"ClusterInformation", f.Name,
	)

	c.mapping[f.Name] = f
}

func (c *clusterInformationController) OnUpdate(oldObj, newObj interface{}) {
	c.mut.Lock()
	defer c.mut.Unlock()

	f := newObj.(*v1alpha1.ClusterInformation)
	f = f.DeepCopy()
	c.logger.Info("OnUpdate",
		"ClusterInformation", f.Name,
	)

	c.mapping[f.Name] = f
}

func (c *clusterInformationController) OnDelete(obj interface{}) {
	c.mut.Lock()
	defer c.mut.Unlock()

	f := obj.(*v1alpha1.ClusterInformation)
	c.logger.Info("OnDelete",
		"ClusterInformation", f.Name,
	)

	delete(c.mapping, f.Name)
}

func (c *clusterInformationController) Get(name string) *v1alpha1.ClusterInformation {
	c.mut.RLock()
	defer c.mut.RUnlock()
	return c.mapping[name]
}

type ClusterInformationGetter interface {
	Get(name string) *v1alpha1.ClusterInformation
}
