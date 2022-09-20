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
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ferryproxy/ferry/pkg/consts"
	"github.com/ferryproxy/ferry/pkg/utils/objref"
	"github.com/ferryproxy/ferry/pkg/utils/trybuffer"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

type AuthorizedController struct {
	mut           sync.Mutex
	ctx           context.Context
	namespace     string
	labelSelector string
	cache         map[string]map[string]string
	clientset     kubernetes.Interface
	logger        logr.Logger
	try           *trybuffer.TryBuffer
}

type AuthorizedControllerConfig struct {
	Namespace     string
	LabelSelector string
	Logger        logr.Logger
	Clientset     kubernetes.Interface
}

func NewAuthorizedController(conf *AuthorizedControllerConfig) *AuthorizedController {
	return &AuthorizedController{
		cache:         map[string]map[string]string{},
		labelSelector: conf.LabelSelector,
		namespace:     conf.Namespace,
		clientset:     conf.Clientset,
		logger:        conf.Logger,
	}
}

func (s *AuthorizedController) Run(ctx context.Context) error {
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

func (s *AuthorizedController) onAdd(obj interface{}) {
	cm := obj.(*corev1.ConfigMap)
	if len(cm.Data) == 0 {
		return
	}

	s.logger.Info("add config map for authorized", "configmap", objref.KObj(cm))
	s.mut.Lock()
	defer s.mut.Unlock()
	s.Add(cm)
}

func (s *AuthorizedController) onUpdate(oldObj, newObj interface{}) {
	cm := newObj.(*corev1.ConfigMap)
	if len(cm.Data) == 0 {
		return
	}

	s.logger.Info("update config map for authorized", "configmap", objref.KObj(cm))
	s.mut.Lock()
	defer s.mut.Unlock()
	s.Add(cm)
}

func (s *AuthorizedController) onDelete(obj interface{}) {
	cm := obj.(*corev1.ConfigMap)
	if len(cm.Data) == 0 {
		return
	}

	s.logger.Info("delete config map for authorized", "configmap", objref.KObj(cm))
	s.mut.Lock()
	defer s.mut.Unlock()
	s.Del(cm)
}

func (s *AuthorizedController) Add(cm *corev1.ConfigMap) {
	authorizedContent := cm.Data[consts.TunnelAuthorizedKeyName]
	user := cm.Data[consts.TunnelUserKey]

	s.cache[cm.Name] = map[string]string{
		user: authorizedContent,
	}

	s.try.Try()
}

func (s *AuthorizedController) Del(cm *corev1.ConfigMap) {
	delete(s.cache, cm.Name)
	s.try.Try()
}

func (s *AuthorizedController) sync() {
	userAuthorized := map[string][]string{}
	for _, authorized := range s.cache {
		for user, auth := range authorized {
			userAuthorized[user] = append(userAuthorized[user], auth)
		}
	}
	for user, authorized := range userAuthorized {

		file := filepath.Join(consts.TunnelSshHomeDir, user, ".ssh", consts.TunnelAuthorizedKeyName)
		err := os.MkdirAll(filepath.Dir(file), 0755)
		if err != nil {
			s.logger.Error(err, "failed to mkdir for authorized config")
			continue
		}
		err = atomicWrite(file, []byte(strings.Join(authorized, "\n")), 0644)
		if err != nil {
			s.logger.Error(err, "failed to write authorized config")
			continue
		}
	}

}
