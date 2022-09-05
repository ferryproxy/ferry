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

package mcs

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"sync"

	"github.com/ferryproxy/api/apis/traffic/v1alpha2"
	ferryversioned "github.com/ferryproxy/client-go/generated/clientset/versioned"
	"github.com/ferryproxy/ferry/pkg/consts"
	"github.com/ferryproxy/ferry/pkg/ferry-controller/router/resource"
	"github.com/ferryproxy/ferry/pkg/utils/diffobjs"
	"github.com/ferryproxy/ferry/pkg/utils/objref"
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	restclient "k8s.io/client-go/rest"
	"sigs.k8s.io/mcs-api/pkg/apis/v1alpha1"
	"sigs.k8s.io/mcs-api/pkg/client/clientset/versioned"
)

type ClusterCache interface {
	ListMCS(namespace string) (map[string][]*v1alpha1.ServiceImport, map[string][]*v1alpha1.ServiceExport)
}

type MCSControllerConfig struct {
	Logger       logr.Logger
	Config       *restclient.Config
	ClusterCache ClusterCache
	Namespace    string
}

type MCSController struct {
	ctx                context.Context
	clientset          versioned.Interface
	ferryClientset     *ferryversioned.Clientset
	config             *restclient.Config
	logger             logr.Logger
	namespace          string
	mut                sync.RWMutex
	clusterCache       ClusterCache
	cacheRoutePolicies []*v1alpha2.RoutePolicy
}

func NewMCSController(conf *MCSControllerConfig) *MCSController {
	return &MCSController{
		config:       conf.Config,
		namespace:    conf.Namespace,
		clusterCache: conf.ClusterCache,
		logger:       conf.Logger,
	}
}

func (m *MCSController) Start(ctx context.Context) error {
	clientset, err := ferryversioned.NewForConfig(m.config)
	if err != nil {
		return err
	}
	m.ferryClientset = clientset

	list, err := m.ferryClientset.
		TrafficV1alpha2().
		RoutePolicies(m.namespace).
		List(ctx, metav1.ListOptions{
			LabelSelector: labels.FormatLabels(labelsForRoutePolicy),
		})
	if err != nil {
		return err
	}
	for _, item := range list.Items {
		m.cacheRoutePolicies = append(m.cacheRoutePolicies, item.DeepCopy())
	}

	return nil
}

func (m *MCSController) Sync(ctx context.Context) {
	m.mut.Lock()
	defer m.mut.Unlock()

	importMap, exportMap := m.clusterCache.ListMCS("")

	updated := mcsToRoutePolicies(importMap, exportMap)

	if reflect.DeepEqual(m.cacheRoutePolicies, updated) {
		m.logger.Info("RoutePolicy not modified")
		return
	}

	m.logger.Info("Update RoutePolicy with mcs", "size", len(updated))

	// Update the cache of RoutePolicy
	deleted := diffobjs.ShouldDeleted(m.cacheRoutePolicies, updated)
	defer func() {
		m.cacheRoutePolicies = updated
	}()

	for _, r := range updated {
		mr := resource.RoutePolicy{r}
		err := mr.Apply(ctx, m.ferryClientset)
		if err != nil {
			m.logger.Error(err, "failed to update RoutePolicy")
		}
	}

	for _, r := range deleted {
		mr := resource.RoutePolicy{r}
		err := mr.Delete(ctx, m.ferryClientset)
		if err != nil {
			m.logger.Error(err, "failed to delete RoutePolicy")
		}
	}
}

func mcsToRoutePolicies(importMap map[string][]*v1alpha1.ServiceImport, exportMap map[string][]*v1alpha1.ServiceExport) []*v1alpha2.RoutePolicy {
	rulesImport := map[objref.ObjectRef][]string{}
	for name, imports := range importMap {
		for _, i := range imports {
			r := objref.ObjectRef{
				Namespace: i.Namespace,
				Name:      i.Name,
			}
			rulesImport[r] = append(rulesImport[r], name)
		}
	}

	rulesExport := map[objref.ObjectRef][]string{}
	for name, exports := range exportMap {
		for _, e := range exports {
			r := objref.ObjectRef{
				Namespace: e.Namespace,
				Name:      e.Name,
			}
			rulesExport[r] = append(rulesExport[r], name)
		}
	}

	policies := []*v1alpha2.RoutePolicy{}
	for n, rule := range rulesImport {
		if len(rulesExport[n]) == 0 {
			continue
		}
		exports := []v1alpha2.RoutePolicySpecRule{}
		for _, r := range rulesExport[n] {
			exports = append(exports, v1alpha2.RoutePolicySpecRule{
				HubName: r,
				Service: v1alpha2.RoutePolicySpecRuleService{
					Namespace: n.Namespace,
					Name:      n.Name,
				},
			})
		}
		if len(exports) == 0 {
			continue
		}

		imports := []v1alpha2.RoutePolicySpecRule{}
		for _, r := range rule {
			imports = append(imports, v1alpha2.RoutePolicySpecRule{
				HubName: r,
				Service: v1alpha2.RoutePolicySpecRuleService{
					Namespace: n.Namespace,
					Name:      n.Name,
				},
			})
		}
		if len(imports) == 0 {
			continue
		}

		sort.Slice(exports, func(i, j int) bool {
			return exports[i].HubName < exports[j].HubName
		})
		sort.Slice(imports, func(i, j int) bool {
			return imports[i].HubName < imports[j].HubName
		})
		policy := v1alpha2.RoutePolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("mcs-%s-%s", n.Namespace, n.Name),
				Namespace: consts.FerryNamespace,
				Labels:    labelsForRoutePolicy,
			},
			Spec: v1alpha2.RoutePolicySpec{
				Exports: exports,
				Imports: imports,
			},
		}
		policies = append(policies, &policy)
	}
	sort.Slice(policies, func(i, j int) bool {
		return policies[i].Name < policies[j].Name
	})
	return policies
}

var labelsForRoutePolicy = map[string]string{
	consts.LabelGeneratedKey: consts.LabelGeneratedValue,
}
