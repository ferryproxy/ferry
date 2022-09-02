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
	"bytes"
	"context"
	"encoding/json"
	"sort"
	"sync"

	"github.com/ferryproxy/ferry/pkg/consts"
	"github.com/ferryproxy/ferry/pkg/utils/objref"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

type ConfigWatcher struct {
	clientset     kubernetes.Interface
	namespace     string
	labelSelector string
	logger        logr.Logger
	mut           sync.Mutex
	cache         map[string][]json.RawMessage
	reloadFunc    func(d []json.RawMessage)
}

type ConfigWatcherConfig struct {
	Clientset     kubernetes.Interface
	Logger        logr.Logger
	Namespace     string
	LabelSelector string
	ReloadFunc    func(d []json.RawMessage)
}

func NewConfigWatcher(conf *ConfigWatcherConfig) *ConfigWatcher {
	n := &ConfigWatcher{
		clientset:     conf.Clientset,
		namespace:     conf.Namespace,
		labelSelector: conf.LabelSelector,
		logger:        conf.Logger,
		cache:         map[string][]json.RawMessage{},
		reloadFunc:    conf.ReloadFunc,
	}
	return n
}

func (c *ConfigWatcher) Run(ctx context.Context) error {
	informer := informers.NewSharedInformerFactoryWithOptions(c.clientset, 0,
		informers.WithNamespace(c.namespace),
		informers.WithTweakListOptions(func(options *metav1.ListOptions) {
			options.LabelSelector = c.labelSelector
		}),
	).Core().V1().ConfigMaps().Informer()
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.onAdd,
		UpdateFunc: c.onUpdate,
		DeleteFunc: c.onDelete,
	})
	informer.Run(ctx.Done())
	return nil
}

func (c *ConfigWatcher) Reload() {
	c.mut.Lock()
	defer c.mut.Unlock()

	sum := []json.RawMessage{}
	uniq := map[string]struct{}{}
	for _, data := range c.cache {
		for _, d := range data {
			if _, ok := uniq[string(d)]; !ok {
				uniq[string(d)] = struct{}{}
				sum = append(sum, d)
			}
		}
	}
	sort.SliceStable(sum, func(i, j int) bool {
		return string(sum[i]) < string(sum[j])
	})

	c.reloadFunc(sum)
}

func (c *ConfigWatcher) update(cm *corev1.ConfigMap) {
	data := make([]json.RawMessage, 0, len(cm.Data)+len(cm.BinaryData))
	content := cm.Data[consts.TunnelRulesKey]

	tmp := []json.RawMessage{}
	err := json.Unmarshal([]byte(content), &tmp)
	if err != nil {
		c.logger.Error(err, "unmarshal context failed",
			"configmap", objref.KObj(cm),
			"context", content,
		)
		return
	}
	for _, item := range tmp {
		v, err := shrinkJSON(item)
		if err != nil {
			c.logger.Error(err, "shrink json failed",
				"configmap", objref.KObj(cm),
				"item", item,
			)
			continue
		}
		data = append(data, v)
	}

	defer c.Reload()
	c.mut.Lock()
	defer c.mut.Unlock()

	c.cache[cm.Name] = data
}

func (c *ConfigWatcher) delete(cm *corev1.ConfigMap) {
	defer c.Reload()
	c.mut.Lock()
	defer c.mut.Unlock()
	delete(c.cache, cm.Name)
}

func (c *ConfigWatcher) onAdd(obj interface{}) {
	cm := obj.(*corev1.ConfigMap)
	c.logger.Info("add configmap for rules", "configmap", objref.KObj(cm))
	c.update(cm)
}

func (c *ConfigWatcher) onUpdate(oldObj, newObj interface{}) {
	cm := newObj.(*corev1.ConfigMap)
	c.logger.Info("update configmap for rules", "configmap", objref.KObj(cm))
	c.update(cm)
}

func (c *ConfigWatcher) onDelete(obj interface{}) {
	cm := obj.(*corev1.ConfigMap)
	c.logger.Info("delete configmap for rules", "configmap", objref.KObj(cm))
	c.delete(cm)
}

func shrinkJSON(src []byte) ([]byte, error) {
	var buf bytes.Buffer
	err := json.Indent(&buf, src, "", "")
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
