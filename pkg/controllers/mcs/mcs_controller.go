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
	"sort"
	"sync"

	trafficv1alpha2 "github.com/ferryproxy/api/apis/traffic/v1alpha2"
	"github.com/ferryproxy/ferry/pkg/client"
	"github.com/ferryproxy/ferry/pkg/consts"
	"github.com/ferryproxy/ferry/pkg/utils/diffobjs"
	"github.com/ferryproxy/ferry/pkg/utils/objref"
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	mcsv1alpha1 "sigs.k8s.io/mcs-api/pkg/apis/v1alpha1"
)

type HubInterface interface {
	ListMCS(namespace string) (map[string][]*mcsv1alpha1.ServiceImport, map[string][]*mcsv1alpha1.ServiceExport)
}

type MCSControllerConfig struct {
	Logger       logr.Logger
	Clientset    client.Interface
	HubInterface HubInterface
	Namespace    string
}

type MCSController struct {
	ctx                context.Context
	clientset          client.Interface
	logger             logr.Logger
	namespace          string
	mut                sync.RWMutex
	hubInterface       HubInterface
	cacheRoutePolicies []*trafficv1alpha2.RoutePolicy
}

func NewMCSController(conf *MCSControllerConfig) *MCSController {
	return &MCSController{
		clientset:    conf.Clientset,
		namespace:    conf.Namespace,
		hubInterface: conf.HubInterface,
		logger:       conf.Logger,
	}
}

func (m *MCSController) Start(ctx context.Context) error {
	list, err := m.clientset.
		Ferry().
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

	importMap, exportMap := m.hubInterface.ListMCS("")

	updated := mcsToRoutePolicies(importMap, exportMap)

	m.logger.Info("Update routePolicy with mcs",
		"size", len(updated),
	)

	// Update the cache of routePolicy
	deleted := diffobjs.ShouldDeleted(m.cacheRoutePolicies, updated)
	defer func() {
		m.cacheRoutePolicies = updated
	}()

	for _, r := range updated {
		err := client.Apply(ctx, m.logger, m.clientset, r)
		if err != nil {
			m.logger.Error(err, "failed to update routePolicy")
		}
	}

	for _, r := range deleted {
		err := client.Delete(ctx, m.logger, m.clientset, r)
		if err != nil {
			m.logger.Error(err, "failed to delete routePolicy")
		}
	}
}

func mcsToRoutePolicies(importMap map[string][]*mcsv1alpha1.ServiceImport, exportMap map[string][]*mcsv1alpha1.ServiceExport) []*trafficv1alpha2.RoutePolicy {
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

	policies := []*trafficv1alpha2.RoutePolicy{}
	for n, rule := range rulesImport {
		if len(rulesExport[n]) == 0 {
			continue
		}
		exports := []trafficv1alpha2.RoutePolicySpecRule{}
		for _, r := range rulesExport[n] {
			exports = append(exports, trafficv1alpha2.RoutePolicySpecRule{
				HubName: r,
				Service: trafficv1alpha2.RoutePolicySpecRuleService{
					Namespace: n.Namespace,
					Name:      n.Name,
				},
			})
		}
		if len(exports) == 0 {
			continue
		}

		imports := []trafficv1alpha2.RoutePolicySpecRule{}
		for _, r := range rule {
			imports = append(imports, trafficv1alpha2.RoutePolicySpecRule{
				HubName: r,
				Service: trafficv1alpha2.RoutePolicySpecRuleService{
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
		policy := trafficv1alpha2.RoutePolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("mcs-%s-%s", n.Namespace, n.Name),
				Namespace: consts.FerryNamespace,
				Labels:    labelsForRoutePolicy,
			},
			Spec: trafficv1alpha2.RoutePolicySpec{
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
