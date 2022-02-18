package controller

import (
	"context"
	"sync"
	"time"

	"github.com/ferry-proxy/ferry/pkg/utils"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
)

type clusterServiceCache struct {
	parentCtx context.Context
	ctx       context.Context
	cancel    context.CancelFunc

	clientset *kubernetes.Clientset
	cache     map[utils.ObjectRef]*corev1.Service
	callback  map[string]func()

	logger logr.Logger
	try    *utils.TryBuffer

	mut sync.Mutex
}

type clusterServiceCacheConfig struct {
	Clientset *kubernetes.Clientset
	Logger    logr.Logger
}

func newClusterServiceCache(conf clusterServiceCacheConfig) *clusterServiceCache {
	c := &clusterServiceCache{
		clientset: conf.Clientset,
		logger:    conf.Logger,
		callback:  map[string]func(){},
		cache:     map[utils.ObjectRef]*corev1.Service{},
	}
	return c
}

func (c *clusterServiceCache) ResetClientset(clientset *kubernetes.Clientset) error {
	c.mut.Lock()
	defer c.mut.Unlock()

	c.cache = map[utils.ObjectRef]*corev1.Service{}
	if c.cancel != nil {
		c.cancel()
	}
	c.ctx, c.cancel = context.WithCancel(c.parentCtx)
	c.clientset = clientset
	informerFactory := informers.NewSharedInformerFactoryWithOptions(c.clientset, 0)
	informer := informerFactory.Core().V1().Services().Informer()
	informer.AddEventHandler(c)
	go informer.Run(c.ctx.Done())
	return nil
}

func (c *clusterServiceCache) Start(ctx context.Context) error {
	c.parentCtx = ctx
	err := c.ResetClientset(c.clientset)
	if err != nil {
		return err
	}
	c.try = utils.NewTryBuffer(c.sync, 1*time.Second)
	return nil
}

func (c *clusterServiceCache) Close() {
	c.try.Close()
	c.cancel()
}

func (c *clusterServiceCache) ForEach(fun func(svc *corev1.Service)) {
	c.mut.Lock()
	defer c.mut.Unlock()

	for _, svc := range c.cache {
		fun(svc)
	}
}

func (c *clusterServiceCache) sync() {
	c.mut.Lock()
	defer c.mut.Unlock()
	for _, cb := range c.callback {
		cb()
	}
}

func (c *clusterServiceCache) RegistryCallback(name string, fun func()) {
	c.mut.Lock()
	defer c.mut.Unlock()
	c.callback[name] = fun
}

func (c *clusterServiceCache) UnregistryCallback(name string) {
	c.mut.Lock()
	defer c.mut.Unlock()
	delete(c.callback, name)
}

func (c *clusterServiceCache) OnAdd(obj interface{}) {
	svc := obj.(*corev1.Service)
	c.logger.Info("OnAdd",
		"Service", utils.KObj(svc),
	)
	svc = svc.DeepCopy()

	c.mut.Lock()
	defer c.mut.Unlock()
	c.cache[utils.KObj(svc)] = svc
	c.try.Try()
}

func (c *clusterServiceCache) OnUpdate(oldObj, newObj interface{}) {
	svc := newObj.(*corev1.Service)
	c.logger.Info("OnUpdate",
		"Service", utils.KObj(svc),
	)
	svc = svc.DeepCopy()

	c.mut.Lock()
	defer c.mut.Unlock()
	c.cache[utils.KObj(svc)] = svc
	c.try.Try()
}

func (c *clusterServiceCache) OnDelete(obj interface{}) {
	svc := obj.(*corev1.Service)
	c.logger.Info("OnDelete",
		"Service", utils.KObj(svc),
	)
	svc = svc.DeepCopy()

	c.mut.Lock()
	defer c.mut.Unlock()
	delete(c.cache, utils.KObj(svc))
	c.try.Try()
}
