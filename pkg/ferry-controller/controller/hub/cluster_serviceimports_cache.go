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
	"sort"
	"sync"
	"time"

	"github.com/ferryproxy/ferry/pkg/utils/objref"
	"github.com/ferryproxy/ferry/pkg/utils/trybuffer"
	"github.com/go-logr/logr"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/mcs-api/pkg/apis/v1alpha1"
	"sigs.k8s.io/mcs-api/pkg/client/clientset/versioned"
	"sigs.k8s.io/mcs-api/pkg/client/informers/externalversions"
)

type clusterServiceImportCache struct {
	parentCtx context.Context
	ctx       context.Context
	cancel    context.CancelFunc

	clientset versioned.Interface
	cache     map[objref.ObjectRef]*v1alpha1.ServiceImport
	syncFunc  func()

	logger logr.Logger
	try    *trybuffer.TryBuffer

	mut sync.RWMutex
}

type clusterServiceImportCacheConfig struct {
	Clientset versioned.Interface
	Logger    logr.Logger
	SyncFunc  func()
}

func newClusterServiceImportCache(conf clusterServiceImportCacheConfig) *clusterServiceImportCache {
	c := &clusterServiceImportCache{
		clientset: conf.Clientset,
		logger:    conf.Logger,
		syncFunc:  conf.SyncFunc,
		cache:     map[objref.ObjectRef]*v1alpha1.ServiceImport{},
	}
	return c
}

func (c *clusterServiceImportCache) ResetClientset(clientset versioned.Interface) error {
	c.mut.Lock()
	defer c.mut.Unlock()

	c.cache = map[objref.ObjectRef]*v1alpha1.ServiceImport{}
	if c.cancel != nil {
		c.cancel()
	}
	c.ctx, c.cancel = context.WithCancel(c.parentCtx)
	c.clientset = clientset
	informerFactory := externalversions.NewSharedInformerFactoryWithOptions(c.clientset, 0)
	informer := informerFactory.
		Multicluster().
		V1alpha1().
		ServiceImports().
		Informer()
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.onAdd,
		UpdateFunc: c.onUpdate,
		DeleteFunc: c.onDelete,
	})

	go informer.Run(c.ctx.Done())
	return nil
}

func (c *clusterServiceImportCache) Start(ctx context.Context) error {
	c.parentCtx = ctx
	c.try = trybuffer.NewTryBuffer(c.syncFunc, time.Second/2)
	err := c.ResetClientset(c.clientset)
	if err != nil {
		return err
	}
	return nil
}

func (c *clusterServiceImportCache) Close() {
	c.try.Close()
	c.cancel()
}

func (c *clusterServiceImportCache) ForEach(fun func(svc *v1alpha1.ServiceImport)) {
	c.mut.RLock()
	defer c.mut.RUnlock()

	for _, svc := range c.cache {
		fun(svc)
	}
}

func (c *clusterServiceImportCache) List() []*v1alpha1.ServiceImport {
	svcs := make([]*v1alpha1.ServiceImport, 0, len(c.cache))
	c.ForEach(func(svc *v1alpha1.ServiceImport) {
		svcs = append(svcs, svc)
	})

	sort.Slice(svcs, func(i, j int) bool {
		return svcs[i].CreationTimestamp.Before(&svcs[j].CreationTimestamp)
	})
	return svcs
}

func (c *clusterServiceImportCache) ListByNamespace(namespace string) []*v1alpha1.ServiceImport {
	if namespace == "" {
		return c.List()
	}
	svcs := make([]*v1alpha1.ServiceImport, 0, len(c.cache))
	c.ForEach(func(svc *v1alpha1.ServiceImport) {
		if svc.Namespace != namespace {
			return
		}
		svcs = append(svcs, svc)
	})

	sort.Slice(svcs, func(i, j int) bool {
		return svcs[i].CreationTimestamp.Before(&svcs[j].CreationTimestamp)
	})
	return svcs
}

func (c *clusterServiceImportCache) onAdd(obj interface{}) {
	svc := obj.(*v1alpha1.ServiceImport)
	c.logger.Info("onAdd",
		"ServiceImport", objref.KObj(svc),
	)
	svc = svc.DeepCopy()

	c.mut.Lock()
	defer c.mut.Unlock()

	c.cache[objref.KObj(svc)] = svc
	c.try.Try()
}

func (c *clusterServiceImportCache) onUpdate(oldObj, newObj interface{}) {
	svc := newObj.(*v1alpha1.ServiceImport)
	c.logger.Info("onUpdate",
		"ServiceImport", objref.KObj(svc),
	)
	svc = svc.DeepCopy()

	c.mut.Lock()
	defer c.mut.Unlock()

	c.cache[objref.KObj(svc)] = svc
}

func (c *clusterServiceImportCache) onDelete(obj interface{}) {
	svc := obj.(*v1alpha1.ServiceImport)
	c.logger.Info("onDelete",
		"ServiceImport", objref.KObj(svc),
	)
	svc = svc.DeepCopy()

	c.mut.Lock()
	defer c.mut.Unlock()

	delete(c.cache, objref.KObj(svc))
	c.try.Try()
}
