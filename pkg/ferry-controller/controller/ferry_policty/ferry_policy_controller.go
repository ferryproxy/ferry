package ferry_policty

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"sync"

	"github.com/ferry-proxy/api/apis/ferry/v1alpha1"
	versioned "github.com/ferry-proxy/client-go/generated/clientset/versioned"
	externalversions "github.com/ferry-proxy/client-go/generated/informers/externalversions"
	"github.com/ferry-proxy/ferry/pkg/ferry-controller/router"
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

type FerryPolicyControllerConfig struct {
	Logger       logr.Logger
	Config       *restclient.Config
	ClusterCache ClusterCache
	Namespace    string
	SyncFunc     func()
}

type FerryPolicyController struct {
	ctx                          context.Context
	mut                          sync.RWMutex
	config                       *restclient.Config
	clientset                    *versioned.Clientset
	clusterCache                 ClusterCache
	cache                        map[string]*v1alpha1.FerryPolicy
	namespace                    string
	logger                       logr.Logger
	cacheFerryPolicyMappingRules []*v1alpha1.MappingRule
	syncFunc                     func()
}

func NewFerryPolicyController(conf FerryPolicyControllerConfig) *FerryPolicyController {
	return &FerryPolicyController{
		config:       conf.Config,
		namespace:    conf.Namespace,
		logger:       conf.Logger,
		clusterCache: conf.ClusterCache,
		syncFunc:     conf.SyncFunc,
		cache:        map[string]*v1alpha1.FerryPolicy{},
	}
}

func (c *FerryPolicyController) list() []*v1alpha1.FerryPolicy {
	var list []*v1alpha1.FerryPolicy
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

func (c *FerryPolicyController) get(name string) *v1alpha1.FerryPolicy {
	return c.cache[name]
}

func (c *FerryPolicyController) Run(ctx context.Context) error {
	c.logger.Info("FerryPolicy controller started")
	defer c.logger.Info("FerryPolicy controller stopped")

	clientset, err := versioned.NewForConfig(c.config)
	if err != nil {
		return err
	}
	c.clientset = clientset
	c.ctx = ctx
	informerFactory := externalversions.NewSharedInformerFactoryWithOptions(clientset, 0,
		externalversions.WithNamespace(c.namespace))
	informer := informerFactory.
		Ferry().
		V1alpha1().
		FerryPolicies().
		Informer()
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.onAdd,
		UpdateFunc: c.onUpdate,
		DeleteFunc: c.onDelete,
	})

	informer.Run(ctx.Done())
	return nil
}

func (c *FerryPolicyController) updateStatus(name string, phase string, ruleCount int) error {
	fp := c.get(name)
	if fp == nil {
		return fmt.Errorf("not found FerryPolicy %s", name)
	}

	fp = fp.DeepCopy()

	fp.Status.LastSynchronizationTimestamp = metav1.Now()
	fp.Status.Phase = phase
	fp.Status.RuleCount = ruleCount

	_, err := c.clientset.
		FerryV1alpha1().
		FerryPolicies(c.namespace).
		UpdateStatus(c.ctx, fp, metav1.UpdateOptions{})
	return err
}

func (c *FerryPolicyController) onAdd(obj interface{}) {
	f := obj.(*v1alpha1.FerryPolicy)
	f = f.DeepCopy()
	c.logger.Info("onAdd",
		"FerryPolicy", objref.KObj(f),
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

func (c *FerryPolicyController) onUpdate(oldObj, newObj interface{}) {
	f := newObj.(*v1alpha1.FerryPolicy)
	f = f.DeepCopy()
	c.logger.Info("onUpdate",
		"FerryPolicy", objref.KObj(f),
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

func (c *FerryPolicyController) onDelete(obj interface{}) {
	f := obj.(*v1alpha1.FerryPolicy)
	c.logger.Info("onDelete",
		"FerryPolicy", objref.KObj(f),
	)

	c.mut.Lock()
	defer c.mut.Unlock()

	delete(c.cache, f.Name)

	c.syncFunc()
}

func (c *FerryPolicyController) Sync(ctx context.Context) {
	c.mut.Lock()
	defer c.mut.Unlock()

	ferryPolicies := c.list()

	mappingRules := policiesToMappingRules(c.clusterCache, ferryPolicies)

	// If the mapping rules are the same, no need to update
	if reflect.DeepEqual(c.cacheFerryPolicyMappingRules, mappingRules) {
		return
	}

	// Update the cache of mapping rules
	updated, deleted := utils.CalculatePatchResources(c.cacheFerryPolicyMappingRules, mappingRules)
	defer func() {
		c.cacheFerryPolicyMappingRules = mappingRules
	}()

	for _, r := range deleted {
		mr := router.MappingRule{r}
		err := mr.Delete(ctx, c.clientset)
		if err != nil {
			c.logger.Error(err, "failed to delete mapping rule")
		}
	}

	for _, r := range updated {
		mr := router.MappingRule{r}
		err := mr.Apply(ctx, c.clientset)
		if err != nil {
			c.logger.Error(err, "failed to update mapping rule")
		}
	}

	for _, policy := range ferryPolicies {
		err := c.updateStatus(policy.Name, "Worked", len(mappingRules))
		if err != nil {
			c.logger.Error(err, "failed to update status")
		}
	}
}

func policiesToMappingRules(clusterCache ClusterCache, policies []*v1alpha1.FerryPolicy) []*v1alpha1.MappingRule {
	out := []*v1alpha1.MappingRule{}
	rules := groupFerryPolicies(policies)
	controller := true
	for exportClusterName, rule := range rules {
		svcs := clusterCache.ListServices(exportClusterName)
		if len(svcs) == 0 {
			continue
		}

		for importClusterName, matches := range rule {
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

					out = append(out, &v1alpha1.MappingRule{
						ObjectMeta: metav1.ObjectMeta{
							Name:      fmt.Sprintf("%s-%s-%s-%s-%s-%s-%s", policy.Name, exportClusterName, exportNamespace, exportName, importClusterName, importNamespace, importName),
							Namespace: policy.Namespace,
							Labels:    policy.Labels,
							OwnerReferences: []metav1.OwnerReference{
								{
									APIVersion: v1alpha1.GroupVersion.String(),
									Kind:       "FerryPolicy",
									Name:       policy.Name,
									UID:        policy.UID,
									Controller: &controller,
								},
							},
						},
						Spec: v1alpha1.MappingRuleSpec{
							Import: v1alpha1.MappingRuleSpecPorts{
								ClusterName: importClusterName,
								Service: v1alpha1.MappingRuleSpecPortsService{
									Name:      importName,
									Namespace: importNamespace,
								},
							},
							Export: v1alpha1.MappingRuleSpecPorts{
								ClusterName: exportClusterName,
								Service: v1alpha1.MappingRuleSpecPortsService{
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

func groupFerryPolicies(policies []*v1alpha1.FerryPolicy) map[string]map[string][]groupFerryPolicy {
	mapping := map[string]map[string][]groupFerryPolicy{}
	for _, policy := range policies {
		for _, rule := range policy.Spec.Rules {
			for _, export := range rule.Exports {
				if export.ClusterName == "" {
					continue
				}
				if _, ok := mapping[export.ClusterName]; !ok {
					mapping[export.ClusterName] = map[string][]groupFerryPolicy{}
				}
				for _, impor := range rule.Imports {
					if impor.ClusterName == "" || impor.ClusterName == export.ClusterName {
						continue
					}
					if _, ok := mapping[export.ClusterName][impor.ClusterName]; !ok {
						mapping[export.ClusterName][impor.ClusterName] = []groupFerryPolicy{}
					}

					matchRule := groupFerryPolicy{
						Policy: policy,
						Export: export.Match,
						Import: impor.Match,
					}
					mapping[export.ClusterName][impor.ClusterName] = append(mapping[export.ClusterName][impor.ClusterName], matchRule)
				}
			}
		}
	}
	return mapping
}

type groupFerryPolicy struct {
	Policy *v1alpha1.FerryPolicy
	Export v1alpha1.FerryPolicySpecRuleMatch
	Import v1alpha1.FerryPolicySpecRuleMatch
}
