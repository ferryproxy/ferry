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

package controller

import (
	"context"
	"sync"
	"time"

	"github.com/ferryproxy/ferry/pkg/consts"
	"github.com/ferryproxy/ferry/pkg/ferry-controller/router/resource"
	"github.com/ferryproxy/ferry/pkg/services"
	"github.com/ferryproxy/ferry/pkg/utils/diffobjs"
	"github.com/ferryproxy/ferry/pkg/utils/objref"
	"github.com/ferryproxy/ferry/pkg/utils/trybuffer"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

type DiscoveryController struct {
	mut           sync.Mutex
	ctx           context.Context
	ips           []string
	namespace     string
	labelSelector string
	cache         map[objref.ObjectRef]map[string][]services.MappingPort
	cacheDiscover []resource.Resourcer
	clientset     kubernetes.Interface
	logger        logr.Logger
	try           *trybuffer.TryBuffer
}

type DiscoveryControllerConfig struct {
	Namespace     string
	LabelSelector string
	Logger        logr.Logger
	Clientset     kubernetes.Interface
}

func NewDiscoveryController(conf *DiscoveryControllerConfig) *DiscoveryController {
	return &DiscoveryController{
		cache:         map[objref.ObjectRef]map[string][]services.MappingPort{},
		labelSelector: conf.LabelSelector,
		namespace:     conf.Namespace,
		clientset:     conf.Clientset,
		logger:        conf.Logger,
	}
}

func (s *DiscoveryController) Run(ctx context.Context) error {
	s.try = trybuffer.NewTryBuffer(func() {
		s.mut.Lock()
		defer s.mut.Unlock()
		s.sync()
	}, time.Second/10)
	informer := informers.NewSharedInformerFactoryWithOptions(s.clientset, 0,
		informers.WithNamespace(s.namespace),
		informers.WithTweakListOptions(func(options *metav1.ListOptions) {
			options.LabelSelector = s.labelSelector
		}),
	).Core().V1().ConfigMaps().Informer()
	s.ctx = ctx
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    s.onAdd,
		UpdateFunc: s.onUpdate,
		DeleteFunc: s.onDelete,
	})
	informer.Run(ctx.Done())
	return nil
}

func (s *DiscoveryController) onAdd(obj interface{}) {
	cm := obj.(*corev1.ConfigMap)
	if len(cm.Data) == 0 {
		return
	}

	s.logger.Info("add config map for service", "configmap", objref.KObj(cm))
	s.mut.Lock()
	defer s.mut.Unlock()
	s.Add(cm)
}

func (s *DiscoveryController) onUpdate(oldObj, newObj interface{}) {
	cm := newObj.(*corev1.ConfigMap)
	if len(cm.Data) == 0 {
		return
	}

	s.logger.Info("update config map for service", "configmap", objref.KObj(cm))
	s.mut.Lock()
	defer s.mut.Unlock()
	s.Add(cm)
}

func (s *DiscoveryController) onDelete(obj interface{}) {
	cm := obj.(*corev1.ConfigMap)
	if len(cm.Data) == 0 {
		return
	}

	s.logger.Info("delete config map for service", "configmap", objref.KObj(cm))
	s.mut.Lock()
	defer s.mut.Unlock()
	s.Del(cm)
}

func (s *DiscoveryController) UpdateIPs(ips []string) {
	s.mut.Lock()
	defer s.mut.Unlock()
	s.logger.Info("update ips", "old", s.ips, "ips", ips)
	s.ips = ips
	s.sync()
}

func (s *DiscoveryController) Add(cm *corev1.ConfigMap) {
	data, err := services.ServiceFrom(cm.Data)
	if err != nil {
		s.logger.Error(err, "ServiceFrom")
		return
	}

	s.add(cm.Name, data.ImportServiceNamespace, data.ImportServiceName, data.Ports)
	s.try.Try()
}

func (s *DiscoveryController) Del(cm *corev1.ConfigMap) {
	data, err := services.ServiceFrom(cm.Data)
	if err != nil {
		s.logger.Error(err, "ServiceFrom")
		return
	}
	s.delete(cm.Name, data.ImportServiceNamespace, data.ImportServiceName)
	s.try.Try()
}

func (s *DiscoveryController) add(export string, namespace, name string, ports []services.MappingPort) {
	svc := objref.ObjectRef{
		Name:      name,
		Namespace: namespace,
	}

	if s.cache[svc] == nil {
		s.cache[svc] = map[string][]services.MappingPort{}
	}

	s.cache[svc][export] = ports
}

func (s *DiscoveryController) delete(export string, namespace, name string) {
	svc := objref.ObjectRef{
		Name:      name,
		Namespace: namespace,
	}

	if s.cache[svc] == nil {
		return
	}

	delete(s.cache[svc], export)
}

var labelsConfigMap = map[string]string{
	consts.LabelGeneratedKey: consts.LabelGeneratedTunnelValue,
}

func (s *DiscoveryController) sync() {
	if len(s.ips) == 0 {
		return
	}
	resources := []resource.Resourcer{}
	for obj, item := range s.cache {
		meta := metav1.ObjectMeta{
			Name:      obj.Name,
			Namespace: obj.Namespace,
			Labels:    labelsConfigMap,
		}
		if len(item) > 0 {
			resources = append(resources, services.BuildServiceDiscovery(meta, s.ips, item)...)
		}
	}

	deleted := diffobjs.ShouldDeleted(s.cacheDiscover, resources)
	defer func() {
		s.cacheDiscover = resources
	}()
	for _, r := range resources {
		err := r.Apply(s.ctx, s.clientset)
		if err != nil {
			s.logger.Error(err, "failed to update")
		}
	}

	for _, r := range deleted {
		err := r.Delete(s.ctx, s.clientset)
		if err != nil {
			s.logger.Error(err, "failed to delete")
		}
	}
}
