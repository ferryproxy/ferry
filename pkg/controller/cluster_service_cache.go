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

	clientset        *kubernetes.Clientset
	cache            map[utils.ObjectRef]*corev1.Service
	callback         map[string]func()
	callbackOnAdd    map[string]func(obj *corev1.Service)
	callbackOnUpdate map[string]func(old, obj *corev1.Service)
	callbackOnDelete map[string]func(obj *corev1.Service)

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
		clientset:        conf.Clientset,
		logger:           conf.Logger,
		callback:         map[string]func(){},
		callbackOnAdd:    map[string]func(obj *corev1.Service){},
		callbackOnUpdate: map[string]func(old *corev1.Service, obj *corev1.Service){},
		callbackOnDelete: map[string]func(obj *corev1.Service){},
		cache:            map[utils.ObjectRef]*corev1.Service{},
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
	informer := informerFactory.
		Core().
		V1().
		Services().
		Informer()
	informer.AddEventHandler(c)
	go informer.Run(c.ctx.Done())
	return nil
}

func (c *clusterServiceCache) Start(ctx context.Context) error {
	c.parentCtx = ctx
	c.try = utils.NewTryBuffer(c.sync, time.Second/2)
	err := c.ResetClientset(c.clientset)
	if err != nil {
		return err
	}
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

func (c *clusterServiceCache) RegistryOnAdd(name string, fun func(obj *corev1.Service)) {
	c.mut.Lock()
	defer c.mut.Unlock()
	c.callbackOnAdd[name] = fun
}

func (c *clusterServiceCache) UnregistryOnAdd(name string) {
	c.mut.Lock()
	defer c.mut.Unlock()
	delete(c.callbackOnAdd, name)
}

func (c *clusterServiceCache) RegistryOnUpdate(name string, fun func(old, obj *corev1.Service)) {
	c.mut.Lock()
	defer c.mut.Unlock()
	c.callbackOnUpdate[name] = fun
}

func (c *clusterServiceCache) UnregistryOnUpdate(name string) {
	c.mut.Lock()
	defer c.mut.Unlock()
	delete(c.callbackOnUpdate, name)
}

func (c *clusterServiceCache) RegistryOnDelete(name string, fun func(obj *corev1.Service)) {
	c.mut.Lock()
	defer c.mut.Unlock()
	c.callbackOnDelete[name] = fun
}

func (c *clusterServiceCache) UnregistryOnDelete(name string) {
	c.mut.Lock()
	defer c.mut.Unlock()
	delete(c.callbackOnDelete, name)
}

func (c *clusterServiceCache) OnAdd(obj interface{}) {
	svc := obj.(*corev1.Service)
	c.logger.Info("OnAdd",
		"Service", utils.KObj(svc),
	)
	svc = svc.DeepCopy()

	c.mut.Lock()
	defer c.mut.Unlock()

	for _, cb := range c.callbackOnAdd {
		cb(svc)
	}

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

	for _, cb := range c.callbackOnUpdate {
		cb(oldObj.(*corev1.Service), svc)
	}

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

	for _, cb := range c.callbackOnDelete {
		cb(svc)
	}

	delete(c.cache, utils.KObj(svc))
	c.try.Try()
}
