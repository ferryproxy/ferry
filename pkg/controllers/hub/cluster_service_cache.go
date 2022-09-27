/*
Copyright 2022 FerryProxy Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package hub

import (
	"context"
	"reflect"
	"sort"
	"sync"
	"time"

	"github.com/ferryproxy/ferry/pkg/client"
	"github.com/ferryproxy/ferry/pkg/utils/objref"
	"github.com/ferryproxy/ferry/pkg/utils/trybuffer"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
)

type clusterServiceCache struct {
	parentCtx context.Context
	ctx       context.Context
	cancel    context.CancelFunc

	clientset client.Interface
	cache     map[objref.ObjectRef]*corev1.Service
	syncFunc  func()

	logger logr.Logger
	try    *trybuffer.TryBuffer

	informer cache.SharedIndexInformer

	mut sync.RWMutex
}

type clusterServiceCacheConfig struct {
	Clientset client.Interface
	Logger    logr.Logger
	SyncFunc  func()
}

func newClusterServiceCache(conf clusterServiceCacheConfig) *clusterServiceCache {
	c := &clusterServiceCache{
		clientset: conf.Clientset,
		logger:    conf.Logger,
		cache:     map[objref.ObjectRef]*corev1.Service{},
		syncFunc:  conf.SyncFunc,
	}
	return c
}

func (c *clusterServiceCache) ResetClientset(clientset client.Interface) error {
	c.mut.Lock()
	defer c.mut.Unlock()

	c.cache = map[objref.ObjectRef]*corev1.Service{}
	if c.cancel != nil {
		c.cancel()
	}
	c.ctx, c.cancel = context.WithCancel(c.parentCtx)
	c.clientset = clientset
	informerFactory := informers.NewSharedInformerFactoryWithOptions(c.clientset.Kubernetes(), 0)
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
	c.informer = informer

	go informer.Run(c.ctx.Done())
	return nil
}

func (c *clusterServiceCache) Start(ctx context.Context) error {
	c.parentCtx = ctx
	c.try = trybuffer.NewTryBuffer(c.sync, time.Second/10)
	err := c.ResetClientset(c.clientset)
	if err != nil {
		return err
	}
	return nil
}

func (c *clusterServiceCache) Close() {
	c.try.Close()
	if c.cancel != nil {
		c.cancel()
	}
}

func (c *clusterServiceCache) ForEach(fun func(svc *corev1.Service)) {
	c.mut.RLock()
	defer c.mut.RUnlock()

	for _, svc := range c.cache {
		fun(svc)
	}
}

func (c *clusterServiceCache) Get(namespace, name string) (*corev1.Service, bool) {
	c.mut.RLock()
	defer c.mut.RUnlock()
	svc, ok := c.cache[objref.KRef(namespace, name)]
	return svc, ok
}

func (c *clusterServiceCache) List() []*corev1.Service {
	svcs := make([]*corev1.Service, 0, len(c.cache))
	c.ForEach(func(svc *corev1.Service) {
		svcs = append(svcs, svc)
	})

	sort.Slice(svcs, func(i, j int) bool {
		return svcs[i].CreationTimestamp.Before(&svcs[j].CreationTimestamp)
	})
	return svcs
}

func (c *clusterServiceCache) sync() {
	c.syncFunc()
}

func (c *clusterServiceCache) onAdd(obj interface{}) {
	svc := obj.(*corev1.Service)
	c.logger.Info("onAdd",
		"service", objref.KObj(svc),
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
		"service", objref.KObj(svc),
	)
	svc = svc.DeepCopy()

	c.mut.Lock()
	defer c.mut.Unlock()

	old := c.cache[objref.KObj(svc)]
	if reflect.DeepEqual(svc.Spec, old.Spec) {
		c.cache[objref.KObj(svc)] = svc
		return
	}
	c.cache[objref.KObj(svc)] = svc

	c.try.Try()
}

func (c *clusterServiceCache) onDelete(obj interface{}) {
	svc := obj.(*corev1.Service)
	c.logger.Info("onDelete",
		"service", objref.KObj(svc),
	)
	svc = svc.DeepCopy()

	c.mut.Lock()
	defer c.mut.Unlock()

	delete(c.cache, objref.KObj(svc))
	c.try.Try()
}
