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

package route_policy

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"sync"

	"github.com/ferryproxy/api/apis/traffic/v1alpha2"
	externalversions "github.com/ferryproxy/client-go/generated/informers/externalversions"
	"github.com/ferryproxy/ferry/pkg/client"
	"github.com/ferryproxy/ferry/pkg/conditions"
	"github.com/ferryproxy/ferry/pkg/consts"
	"github.com/ferryproxy/ferry/pkg/utils/diffobjs"
	"github.com/ferryproxy/ferry/pkg/utils/maps"
	"github.com/ferryproxy/ferry/pkg/utils/objref"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
)

type HubInterface interface {
	ListHubs() []*v1alpha2.Hub
	ListServices(hubName string) []*corev1.Service
}

type RoutePolicyControllerConfig struct {
	Logger       logr.Logger
	Clientset    client.Interface
	HubInterface HubInterface
	Namespace    string
	SyncFunc     func()
}

type RoutePolicyController struct {
	ctx                    context.Context
	mut                    sync.RWMutex
	mutStatus              sync.Mutex
	clientset              client.Interface
	hubInterface           HubInterface
	cache                  map[string]*v1alpha2.RoutePolicy
	namespace              string
	logger                 logr.Logger
	cacheRoutePolicyRoutes []*v1alpha2.Route
	syncFunc               func()
	conditionsManager      *conditions.ConditionsManager
}

func NewRoutePolicyController(conf RoutePolicyControllerConfig) *RoutePolicyController {
	return &RoutePolicyController{
		clientset:         conf.Clientset,
		namespace:         conf.Namespace,
		logger:            conf.Logger,
		hubInterface:      conf.HubInterface,
		syncFunc:          conf.SyncFunc,
		cache:             map[string]*v1alpha2.RoutePolicy{},
		conditionsManager: conditions.NewConditionsManager(),
	}
}

func (c *RoutePolicyController) list() []*v1alpha2.RoutePolicy {
	var list []*v1alpha2.RoutePolicy
	for _, v := range c.cache {
		item := c.cache[v.Name]
		if item == nil {
			continue
		}
		list = append(list, item)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].Name < list[j].Name
	})
	return list
}

func (c *RoutePolicyController) get(name string) *v1alpha2.RoutePolicy {
	return c.cache[name]
}

func (c *RoutePolicyController) Run(ctx context.Context) error {
	c.logger.Info("routePolicy controller started")
	defer c.logger.Info("routePolicy controller stopped")

	c.ctx = ctx

	list, err := c.clientset.
		Ferry().
		TrafficV1alpha2().
		Routes(c.namespace).
		List(ctx, metav1.ListOptions{
			LabelSelector: labels.FormatLabels(labelsForRoute),
		})
	if err != nil {
		return err
	}
	for _, item := range list.Items {
		c.cacheRoutePolicyRoutes = append(c.cacheRoutePolicyRoutes, item.DeepCopy())
	}

	informerFactory := externalversions.NewSharedInformerFactoryWithOptions(c.clientset.Ferry(), 0,
		externalversions.WithNamespace(c.namespace))
	informer := informerFactory.
		Traffic().
		V1alpha2().
		RoutePolicies().
		Informer()
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.onAdd,
		UpdateFunc: c.onUpdate,
		DeleteFunc: c.onDelete,
	})

	informer.Run(ctx.Done())
	return nil
}

func (c *RoutePolicyController) UpdateRoutePolicyCondition(name string, routeCount int) error {
	c.mutStatus.Lock()
	defer c.mutStatus.Unlock()
	fp := c.get(name)
	if fp == nil {
		return fmt.Errorf("not found routePolicy %s", name)
	}

	status := fp.Status.DeepCopy()

	if routeCount > 0 {
		c.conditionsManager.Set(name, metav1.Condition{
			Type:   v1alpha2.RoutePolicyReady,
			Status: metav1.ConditionTrue,
			Reason: v1alpha2.RoutePolicyReady,
		})
		status.Phase = v1alpha2.RoutePolicyReady
	} else {
		c.conditionsManager.Set(name, metav1.Condition{
			Type:   v1alpha2.RoutePolicyReady,
			Status: metav1.ConditionTrue,
			Reason: "NotReady",
		})
		status.Phase = "NotReady"
	}

	status.LastSynchronizationTimestamp = metav1.Now()
	status.RouteCount = routeCount
	status.Conditions = c.conditionsManager.Get(name)

	data, err := json.Marshal(map[string]interface{}{
		"status": status,
	})
	if err != nil {
		return err
	}
	_, err = c.clientset.
		Ferry().
		TrafficV1alpha2().
		RoutePolicies(fp.Namespace).
		Patch(c.ctx, fp.Name, types.MergePatchType, data, metav1.PatchOptions{}, "status")
	return err
}

func (c *RoutePolicyController) onAdd(obj interface{}) {
	f := obj.(*v1alpha2.RoutePolicy)
	f = f.DeepCopy()
	c.logger.Info("onAdd",
		"routePolicy", objref.KObj(f),
	)

	c.mut.Lock()
	defer c.mut.Unlock()

	c.cache[f.Name] = f

	c.syncFunc()

	err := c.UpdateRoutePolicyCondition(f.Name, 0)
	if err != nil {
		c.logger.Error(err, "failed to update status",
			"routePolicy", objref.KObj(f),
		)
	}
}

func (c *RoutePolicyController) onUpdate(oldObj, newObj interface{}) {
	f := newObj.(*v1alpha2.RoutePolicy)
	f = f.DeepCopy()
	c.logger.Info("onUpdate",
		"routePolicy", objref.KObj(f),
	)

	c.mut.Lock()
	defer c.mut.Unlock()

	if reflect.DeepEqual(c.cache[f.Name].Spec, f.Spec) {
		c.cache[f.Name] = f
		return
	}

	c.cache[f.Name] = f

	c.syncFunc()
}

func (c *RoutePolicyController) onDelete(obj interface{}) {
	f := obj.(*v1alpha2.RoutePolicy)
	c.logger.Info("onDelete",
		"routePolicy", objref.KObj(f),
	)

	c.mut.Lock()
	defer c.mut.Unlock()

	delete(c.cache, f.Name)

	c.conditionsManager.Delete(f.Name)
	c.syncFunc()
}

func (c *RoutePolicyController) Sync(ctx context.Context) {
	c.mut.Lock()
	defer c.mut.Unlock()

	ferryPolicies := c.list()

	updated := policiesToRoutes(c.hubInterface, ferryPolicies)

	hubs := c.hubInterface.ListHubs()

	routes := BuildMirrorTunnelRoutes(hubs, consts.ControlPlaneName)
	updated = append(updated, routes...)

	// If the mapping rules are the same, no need to update
	if reflect.DeepEqual(c.cacheRoutePolicyRoutes, updated) {
		return
	}

	// Update the cache of mapping rules
	deleted := diffobjs.ShouldDeleted(c.cacheRoutePolicyRoutes, updated)
	defer func() {
		c.cacheRoutePolicyRoutes = updated
	}()

	for _, r := range deleted {
		err := client.Delete(ctx, c.clientset, r)
		if err != nil {
			c.logger.Error(err, "failed to delete mapping rule")
		}
	}

	for _, r := range updated {
		err := client.Apply(ctx, c.clientset, r)
		if err != nil {
			c.logger.Error(err, "failed to update mapping rule")
		}
	}

	for _, policy := range ferryPolicies {
		count := 0

		for _, r := range updated {
			if policy.UID == r.OwnerReferences[0].UID {
				count++
			}
		}
		err := c.UpdateRoutePolicyCondition(policy.Name, count)
		if err != nil {
			c.logger.Error(err, "failed to update status",
				"routePolicy", objref.KObj(policy),
			)
		}
	}
}

func policiesToRoutes(hubInterface HubInterface, policies []*v1alpha2.RoutePolicy) []*v1alpha2.Route {
	out := []*v1alpha2.Route{}
	rules := groupFerryPolicies(policies)
	controller := true
	for exportHubName, rule := range rules {
		svcs := hubInterface.ListServices(exportHubName)
		if len(svcs) == 0 {
			continue
		}

		for importHubName, matches := range rule {
			for _, match := range matches {
				label := maps.Merge(match.Export.Labels, match.Import.Labels)
				var labelsMatch labels.Selector

				for _, svc := range svcs {
					var (
						exportName      = match.Export.Name
						exportNamespace = match.Export.Namespace
						importName      = match.Import.Name
						importNamespace = match.Import.Namespace
					)

					if exportName == "" && importName != "" {
						exportName = importName
					}
					if exportNamespace == "" && importNamespace != "" {
						exportNamespace = importNamespace
					}

					if importName == "" && exportName != "" {
						importName = exportName
					}
					if importNamespace == "" && exportNamespace != "" {
						importNamespace = exportNamespace
					}

					if len(label) != 0 && exportName == "" {
						if exportNamespace != "" && exportNamespace != svc.Namespace {
							continue
						}

						if labelsMatch == nil {
							labelsMatch = labels.SelectorFromSet(label)
						}
						if !labelsMatch.Matches(labels.Set(svc.Labels)) {
							continue
						}

						exportNamespace = svc.Namespace

						exportName = svc.Name

					} else {
						if exportNamespace == "" {
							continue
						}

						if svc.Namespace != exportNamespace {
							continue
						}

						if exportName != "" && svc.Name != exportName {
							continue
						}
						if exportName == "" {
							exportName = svc.Name
						}
					}

					if importName == "" {
						importName = exportName
					}

					if importNamespace == "" {
						importNamespace = exportNamespace
					}

					policy := match.Policy

					suffix := hash(fmt.Sprintf("%s|%s|%s|%s|%s|%s",
						exportHubName, exportNamespace, exportName,
						importHubName, importNamespace, importName))
					out = append(out, &v1alpha2.Route{
						ObjectMeta: metav1.ObjectMeta{
							Name:      fmt.Sprintf("%s-%s", policy.Name, suffix),
							Namespace: policy.Namespace,
							Labels:    maps.Merge(policy.Labels, labelsForRoute),
							OwnerReferences: []metav1.OwnerReference{
								{
									APIVersion: v1alpha2.GroupVersion.String(),
									Kind:       "RoutePolicy",
									Name:       policy.Name,
									UID:        policy.UID,
									Controller: &controller,
								},
							},
						},
						Spec: v1alpha2.RouteSpec{
							Import: v1alpha2.RouteSpecRule{
								HubName: importHubName,
								Service: v1alpha2.RouteSpecRuleService{
									Name:      importName,
									Namespace: importNamespace,
								},
							},
							Export: v1alpha2.RouteSpecRule{
								HubName: exportHubName,
								Service: v1alpha2.RouteSpecRuleService{
									Name:      exportName,
									Namespace: exportNamespace,
								},
							},
						},
					})
				}
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out
}

func groupFerryPolicies(policies []*v1alpha2.RoutePolicy) map[string]map[string][]groupRoutePolicy {
	mapping := map[string]map[string][]groupRoutePolicy{}
	for _, policy := range policies {

		for _, export := range policy.Spec.Exports {
			if export.HubName == "" {
				continue
			}
			if _, ok := mapping[export.HubName]; !ok {
				mapping[export.HubName] = map[string][]groupRoutePolicy{}
			}
			for _, impor := range policy.Spec.Imports {
				if impor.HubName == "" || impor.HubName == export.HubName {
					continue
				}
				if _, ok := mapping[export.HubName][impor.HubName]; !ok {
					mapping[export.HubName][impor.HubName] = []groupRoutePolicy{}
				}

				matchRule := groupRoutePolicy{
					Policy: policy,
					Export: export.Service,
					Import: impor.Service,
				}
				mapping[export.HubName][impor.HubName] = append(mapping[export.HubName][impor.HubName], matchRule)
			}
		}
	}
	return mapping
}

type groupRoutePolicy struct {
	Policy *v1alpha2.RoutePolicy
	Export v1alpha2.RoutePolicySpecRuleService
	Import v1alpha2.RoutePolicySpecRuleService
}

var labelsForRoute = map[string]string{
	consts.LabelGeneratedKey: consts.LabelGeneratedValue,
}

func hash(s string) string {
	d := sha256.Sum256([]byte(s))
	return hex.EncodeToString(d[:6])
}
