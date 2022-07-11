package route_policty

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"sync"

	"github.com/ferry-proxy/api/apis/traffic/v1alpha2"
	versioned "github.com/ferry-proxy/client-go/generated/clientset/versioned"
	externalversions "github.com/ferry-proxy/client-go/generated/informers/externalversions"
	"github.com/ferry-proxy/ferry/pkg/ferry-controller/router/resource"
	"github.com/ferry-proxy/ferry/pkg/ferry-controller/utils"
	"github.com/ferry-proxy/ferry/pkg/utils/objref"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

type ClusterCache interface {
	ListServices(name string) []*corev1.Service
}

type RoutePolicyControllerConfig struct {
	Logger       logr.Logger
	Config       *restclient.Config
	ClusterCache ClusterCache
	Namespace    string
	SyncFunc     func()
}

type RoutePolicyController struct {
	ctx                    context.Context
	mut                    sync.RWMutex
	config                 *restclient.Config
	clientset              *versioned.Clientset
	clusterCache           ClusterCache
	cache                  map[string]*v1alpha2.RoutePolicy
	namespace              string
	logger                 logr.Logger
	cacheRoutePolicyRoutes []*v1alpha2.Route
	syncFunc               func()
}

func NewRoutePolicyController(conf RoutePolicyControllerConfig) *RoutePolicyController {
	return &RoutePolicyController{
		config:       conf.Config,
		namespace:    conf.Namespace,
		logger:       conf.Logger,
		clusterCache: conf.ClusterCache,
		syncFunc:     conf.SyncFunc,
		cache:        map[string]*v1alpha2.RoutePolicy{},
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
	c.logger.Info("RoutePolicy controller started")
	defer c.logger.Info("RoutePolicy controller stopped")

	clientset, err := versioned.NewForConfig(c.config)
	if err != nil {
		return err
	}
	c.clientset = clientset
	c.ctx = ctx
	informerFactory := externalversions.NewSharedInformerFactoryWithOptions(clientset, 0,
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

func (c *RoutePolicyController) updateStatus(name string, phase string, routeCount int) error {
	fp := c.get(name)
	if fp == nil {
		return fmt.Errorf("not found RoutePolicy %s", name)
	}

	fp = fp.DeepCopy()

	fp.Status.LastSynchronizationTimestamp = metav1.Now()
	fp.Status.Phase = phase
	fp.Status.RouteCount = routeCount

	_, err := c.clientset.
		TrafficV1alpha2().
		RoutePolicies(c.namespace).
		UpdateStatus(c.ctx, fp, metav1.UpdateOptions{})
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

	err := c.updateStatus(f.Name, "Pending", 0)
	if err != nil {
		c.logger.Error(err, "failed to update status")
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

	err := c.updateStatus(f.Name, "Pending", 0)
	if err != nil {
		c.logger.Error(err, "failed to update status")
	}
}

func (c *RoutePolicyController) onDelete(obj interface{}) {
	f := obj.(*v1alpha2.RoutePolicy)
	c.logger.Info("onDelete",
		"routePolicy", objref.KObj(f),
	)

	c.mut.Lock()
	defer c.mut.Unlock()

	delete(c.cache, f.Name)

	c.syncFunc()
}

func (c *RoutePolicyController) Sync(ctx context.Context) {
	c.mut.Lock()
	defer c.mut.Unlock()

	ferryPolicies := c.list()

	routes := policiesToRoutes(c.clusterCache, ferryPolicies)

	// If the mapping rules are the same, no need to update
	if reflect.DeepEqual(c.cacheRoutePolicyRoutes, routes) {
		return
	}

	// Update the cache of mapping rules
	updated, deleted := utils.CalculatePatchResources(c.cacheRoutePolicyRoutes, routes)
	defer func() {
		c.cacheRoutePolicyRoutes = routes
	}()

	for _, r := range deleted {
		mr := resource.Route{r}
		err := mr.Delete(ctx, c.clientset)
		if err != nil {
			c.logger.Error(err, "failed to delete mapping rule")
		}
	}

	for _, r := range updated {
		mr := resource.Route{r}
		err := mr.Apply(ctx, c.clientset)
		if err != nil {
			c.logger.Error(err, "failed to update mapping rule")
		}
	}

	for _, policy := range ferryPolicies {
		err := c.updateStatus(policy.Name, "Worked", len(routes))
		if err != nil {
			c.logger.Error(err, "failed to update status")
		}
	}
}

func policiesToRoutes(clusterCache ClusterCache, policies []*v1alpha2.RoutePolicy) []*v1alpha2.Route {
	out := []*v1alpha2.Route{}
	rules := groupFerryPolicies(policies)
	controller := true
	for exportHubName, rule := range rules {
		svcs := clusterCache.ListServices(exportHubName)
		if len(svcs) == 0 {
			continue
		}

		for importHubName, matches := range rule {
			for _, match := range matches {
				for _, svc := range svcs {
					var (
						exportName      = match.Export.Name
						exportNamespace = match.Export.Namespace
						importName      = match.Import.Name
						importNamespace = match.Import.Namespace
					)

					if len(match.Export.Labels) != 0 {
						if !labels.SelectorFromSet(match.Export.Labels).Matches(labels.Set(svc.Labels)) {
							continue
						}
						if exportName == "" {
							exportName = svc.Name
						}
						if exportNamespace == "" {
							exportNamespace = svc.Namespace
						}
					} else {
						if svc.Namespace != exportNamespace {
							continue
						}

						if svc.Name != exportName {
							continue
						}
					}

					if importName == "" {
						importName = exportName
					}

					if importNamespace == "" {
						importNamespace = exportNamespace
					}

					policy := match.Policy

					out = append(out, &v1alpha2.Route{
						ObjectMeta: metav1.ObjectMeta{
							Name:      fmt.Sprintf("%s-%s-%s-%s-%s-%s-%s", policy.Name, exportHubName, exportNamespace, exportName, importHubName, importNamespace, importName),
							Namespace: policy.Namespace,
							Labels:    policy.Labels,
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
