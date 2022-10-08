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

	"github.com/ferryproxy/ferry/pkg/client"
	"github.com/ferryproxy/ferry/pkg/utils/objref"
	"github.com/ferryproxy/ferry/pkg/utils/trybuffer"
	"github.com/go-logr/logr"
	"k8s.io/client-go/tools/cache"
	mcsv1alpha1 "sigs.k8s.io/mcs-api/pkg/apis/v1alpha1"
	"sigs.k8s.io/mcs-api/pkg/client/informers/externalversions"
)

type clusterServiceExportCache struct {
	parentCtx context.Context
	ctx       context.Context
	cancel    context.CancelFunc

	clientset client.Interface
	cache     map[objref.ObjectRef]*mcsv1alpha1.ServiceExport
	syncFunc  func()

	logger logr.Logger
	try    *trybuffer.TryBuffer

	informer cache.SharedIndexInformer

	mut sync.RWMutex
}

type clusterServiceExportCacheConfig struct {
	Clientset client.Interface
	Logger    logr.Logger
	SyncFunc  func()
}

func newClusterServiceExportCache(conf clusterServiceExportCacheConfig) *clusterServiceExportCache {
	c := &clusterServiceExportCache{
		clientset: conf.Clientset,
		logger:    conf.Logger,
		syncFunc:  conf.SyncFunc,
		cache:     map[objref.ObjectRef]*mcsv1alpha1.ServiceExport{},
	}
	return c
}

func (c *clusterServiceExportCache) ResetClientset(clientset client.Interface) error {
	c.mut.Lock()
	defer c.mut.Unlock()

	c.cache = map[objref.ObjectRef]*mcsv1alpha1.ServiceExport{}
	if c.cancel != nil {
		c.cancel()
	}
	c.ctx, c.cancel = context.WithCancel(c.parentCtx)
	c.clientset = clientset
	informerFactory := externalversions.NewSharedInformerFactoryWithOptions(c.clientset.MCS(), 0)
	informer := informerFactory.
		Multicluster().
		V1alpha1().
		ServiceExports().
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

func (c *clusterServiceExportCache) Start(ctx context.Context) error {
	c.parentCtx = ctx
	c.try = trybuffer.NewTryBuffer(c.syncFunc, time.Second/10)
	err := c.ResetClientset(c.clientset)
	if err != nil {
		return err
	}
	return nil
}

func (c *clusterServiceExportCache) Close() {
	c.try.Close()
	c.cancel()
}

func (c *clusterServiceExportCache) ForEach(fun func(svc *mcsv1alpha1.ServiceExport)) {
	c.mut.RLock()
	defer c.mut.RUnlock()

	for _, svc := range c.cache {
		fun(svc)
	}
}

func (c *clusterServiceExportCache) List() []*mcsv1alpha1.ServiceExport {
	svcs := make([]*mcsv1alpha1.ServiceExport, 0, len(c.cache))
	c.ForEach(func(svc *mcsv1alpha1.ServiceExport) {
		svcs = append(svcs, svc)
	})

	sort.Slice(svcs, func(i, j int) bool {
		return svcs[i].CreationTimestamp.Before(&svcs[j].CreationTimestamp)
	})
	return svcs
}

func (c *clusterServiceExportCache) ListByNamespace(namespace string) []*mcsv1alpha1.ServiceExport {
	if namespace == "" {
		return c.List()
	}
	svcs := make([]*mcsv1alpha1.ServiceExport, 0, len(c.cache))
	c.ForEach(func(svc *mcsv1alpha1.ServiceExport) {
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

func (c *clusterServiceExportCache) onAdd(obj interface{}) {
	svc := obj.(*mcsv1alpha1.ServiceExport)
	c.logger.Info("onAdd",
		"serviceExport", objref.KObj(svc),
	)
	svc = svc.DeepCopy()

	c.mut.Lock()
	defer c.mut.Unlock()

	c.cache[objref.KObj(svc)] = svc
	c.try.Try()
}

func (c *clusterServiceExportCache) onUpdate(oldObj, newObj interface{}) {
	svc := newObj.(*mcsv1alpha1.ServiceExport)
	c.logger.Info("onUpdate",
		"serviceExport", objref.KObj(svc),
	)
	svc = svc.DeepCopy()

	c.mut.Lock()
	defer c.mut.Unlock()

	c.cache[objref.KObj(svc)] = svc
}

func (c *clusterServiceExportCache) onDelete(obj interface{}) {
	svc := obj.(*mcsv1alpha1.ServiceExport)
	c.logger.Info("onDelete",
		"serviceExport", objref.KObj(svc),
	)
	svc = svc.DeepCopy()

	c.mut.Lock()
	defer c.mut.Unlock()

	delete(c.cache, objref.KObj(svc))
	c.try.Try()
}
