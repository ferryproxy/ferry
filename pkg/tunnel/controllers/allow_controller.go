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

package controllers

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/ferryproxy/ferry/pkg/consts"
	"github.com/ferryproxy/ferry/pkg/router"
	"github.com/ferryproxy/ferry/pkg/utils/objref"
	"github.com/ferryproxy/ferry/pkg/utils/trybuffer"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

type AllowController struct {
	mut           sync.Mutex
	ctx           context.Context
	namespace     string
	labelSelector string
	cache         map[string]map[string]*router.AllowList
	clientset     kubernetes.Interface
	logger        logr.Logger
	try           *trybuffer.TryBuffer
}

type AllowControllerConfig struct {
	Namespace     string
	LabelSelector string
	Logger        logr.Logger
	Clientset     kubernetes.Interface
}

func NewAllowController(conf *AllowControllerConfig) *AllowController {
	return &AllowController{
		cache:         map[string]map[string]*router.AllowList{},
		labelSelector: conf.LabelSelector,
		namespace:     conf.Namespace,
		clientset:     conf.Clientset,
		logger:        conf.Logger,
	}
}

func (s *AllowController) Run(ctx context.Context) error {
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

func (s *AllowController) onAdd(obj interface{}) {
	cm := obj.(*corev1.ConfigMap)
	if len(cm.Data) == 0 {
		return
	}

	s.logger.Info("add config map for allows",
		"configMap", objref.KObj(cm),
	)
	s.mut.Lock()
	defer s.mut.Unlock()
	s.Add(cm)
}

func (s *AllowController) onUpdate(oldObj, newObj interface{}) {
	cm := newObj.(*corev1.ConfigMap)
	if len(cm.Data) == 0 {
		return
	}

	s.logger.Info("update config map for allows",
		"configMap", objref.KObj(cm),
	)
	s.mut.Lock()
	defer s.mut.Unlock()
	s.Add(cm)
}

func (s *AllowController) onDelete(obj interface{}) {
	cm := obj.(*corev1.ConfigMap)
	if len(cm.Data) == 0 {
		return
	}

	s.logger.Info("delete config map for allows",
		"configMap", objref.KObj(cm),
	)
	s.mut.Lock()
	defer s.mut.Unlock()
	s.Del(cm)
}

func (s *AllowController) Add(cm *corev1.ConfigMap) {
	allowData := map[string]*router.AllowList{}
	allowContent := cm.Data[consts.TunnelRulesAllowKey]
	err := json.Unmarshal([]byte(allowContent), &allowData)
	if err != nil {
		s.logger.Error(err, "unmarshal context failed",
			"configMap", objref.KObj(cm),
			"context", allowContent,
		)
		return
	}
	s.cache[cm.Name] = allowData

	s.try.Try()
}

func (s *AllowController) Del(cm *corev1.ConfigMap) {
	delete(s.cache, cm.Name)
	s.try.Try()
}

func (s *AllowController) sync() {
	userAllow := map[string]*router.AllowList{}
	for _, allows := range s.cache {
		for user, allow := range allows {
			a := userAllow[user]
			userAllow[user] = a.Merge(allow)
		}
	}
	for user, allow := range userAllow {
		allowConfig, err := json.Marshal(allow)
		if err != nil {
			s.logger.Error(err, "failed to marshal allow config")
			continue
		}
		file := filepath.Join(consts.TunnelSshHomeDir, user, ".ssh", consts.TunnelPermissionsName)
		err = os.MkdirAll(filepath.Dir(file), 0755)
		if err != nil {
			s.logger.Error(err, "failed to mkdir for allow config")
			continue
		}
		err = atomicWrite(file, allowConfig, 0644)
		if err != nil {
			s.logger.Error(err, "failed to write allow config")
			continue
		}
	}

}
