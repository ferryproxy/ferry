package hub

import (
	"context"
	"sync"
	"time"

	"github.com/ferryproxy/ferry/pkg/utils/objref"
	"github.com/ferryproxy/ferry/pkg/utils/trybuffer"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

type clusterServiceCache struct {
	parentCtx context.Context
	ctx       context.Context
	cancel    context.CancelFunc

	clientset kubernetes.Interface
	cache     map[objref.ObjectRef]*corev1.Service
	callback  map[string]func()

	logger logr.Logger
	try    *trybuffer.TryBuffer

	mut sync.Mutex
}

type clusterServiceCacheConfig struct {
	Clientset kubernetes.Interface
	Logger    logr.Logger
}

func newClusterServiceCache(conf clusterServiceCacheConfig) *clusterServiceCache {
	c := &clusterServiceCache{
		clientset: conf.Clientset,
		logger:    conf.Logger,
		callback:  map[string]func(){},
		cache:     map[objref.ObjectRef]*corev1.Service{},
	}
	return c
}

func (c *clusterServiceCache) ResetClientset(clientset kubernetes.Interface) error {
	c.mut.Lock()
	defer c.mut.Unlock()

	c.cache = map[objref.ObjectRef]*corev1.Service{}
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
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.onAdd,
		UpdateFunc: c.onUpdate,
		DeleteFunc: c.onDelete,
	})

	go informer.Run(c.ctx.Done())
	return nil
}

func (c *clusterServiceCache) Start(ctx context.Context) error {
	c.parentCtx = ctx
	c.try = trybuffer.NewTryBuffer(c.sync, time.Second/2)
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

func (c *clusterServiceCache) onAdd(obj interface{}) {
	svc := obj.(*corev1.Service)
	c.logger.Info("onAdd",
		"Service", objref.KObj(svc),
	)
	svc = svc.DeepCopy()

	c.mut.Lock()
	defer c.mut.Unlock()

	c.cache[objref.KObj(svc)] = svc
	c.try.Try()
}

func (c *clusterServiceCache) onUpdate(oldObj, newObj interface{}) {
	svc := newObj.(*corev1.Service)
	c.logger.Info("onUpdate",
		"Service", objref.KObj(svc),
	)
	svc = svc.DeepCopy()

	c.mut.Lock()
	defer c.mut.Unlock()

	c.cache[objref.KObj(svc)] = svc
	c.try.Try()
}

func (c *clusterServiceCache) onDelete(obj interface{}) {
	svc := obj.(*corev1.Service)
	c.logger.Info("onDelete",
		"Service", objref.KObj(svc),
	)
	svc = svc.DeepCopy()

	c.mut.Lock()
	defer c.mut.Unlock()

	delete(c.cache, objref.KObj(svc))
	c.try.Try()
}
