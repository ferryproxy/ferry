package mapping_rule

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	"github.com/ferry-proxy/api/apis/ferry/v1alpha1"
	versioned "github.com/ferry-proxy/client-go/generated/clientset/versioned"
	externalversions "github.com/ferry-proxy/client-go/generated/informers/externalversions"
	"github.com/ferry-proxy/ferry/pkg/consts"
	"github.com/ferry-proxy/ferry/pkg/controller/cluster_information"
	"github.com/ferry-proxy/ferry/pkg/router"
	original "github.com/ferry-proxy/ferry/pkg/router/tunnel"
	"github.com/ferry-proxy/ferry/pkg/utils"
	"github.com/ferry-proxy/utils/objref"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

type MappingRuleControllerConfig struct {
	Logger                       logr.Logger
	Config                       *restclient.Config
	ClusterInformationController *cluster_information.ClusterInformationController
	Namespace                    string
	SyncFunc                     func()
}

type MappingRuleController struct {
	ctx                          context.Context
	mut                          sync.RWMutex
	config                       *restclient.Config
	clientset                    *versioned.Clientset
	clusterInformationController *cluster_information.ClusterInformationController
	cache                        map[string]*v1alpha1.MappingRule
	cacheDataPlaneController     map[ClusterPair]*dataPlaneController
	cacheMappingRules            map[string]map[string][]*v1alpha1.MappingRule
	namespace                    string
	syncFunc                     func()
	logger                       logr.Logger
}

func NewMappingRuleController(conf *MappingRuleControllerConfig) *MappingRuleController {
	return &MappingRuleController{
		config:                       conf.Config,
		namespace:                    conf.Namespace,
		clusterInformationController: conf.ClusterInformationController,
		logger:                       conf.Logger,
		syncFunc:                     conf.SyncFunc,
		cache:                        map[string]*v1alpha1.MappingRule{},
		cacheDataPlaneController:     map[ClusterPair]*dataPlaneController{},
		cacheMappingRules:            map[string]map[string][]*v1alpha1.MappingRule{},
	}
}

func (c *MappingRuleController) List() []*v1alpha1.MappingRule {
	c.mut.RLock()
	defer c.mut.RUnlock()
	var list []*v1alpha1.MappingRule
	for _, v := range c.cache {
		item := c.cache[v.Name]
		if item == nil {
			continue
		}
		list = append(list, item)
	}
	return list
}

func (c *MappingRuleController) Get(name string) *v1alpha1.MappingRule {
	c.mut.RLock()
	defer c.mut.RUnlock()
	return c.cache[name]
}

func (c *MappingRuleController) Run(ctx context.Context) error {
	c.logger.Info("MappingRule controller started")
	defer c.logger.Info("MappingRule controller stopped")

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
		MappingRules().
		Informer()
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.onAdd,
		UpdateFunc: c.onUpdate,
		DeleteFunc: c.onDelete,
	})

	informer.Run(ctx.Done())
	return nil
}

func (c *MappingRuleController) UpdateStatus(name string, phase string) error {
	c.mut.RLock()
	defer c.mut.RUnlock()

	fp := c.cache[name]
	if fp == nil {
		return fmt.Errorf("not found MappingRule %s", name)
	}

	fp = fp.DeepCopy()

	fp.Status.LastSynchronizationTimestamp = metav1.Now()
	fp.Status.Phase = phase
	_, err := c.clientset.
		FerryV1alpha1().
		MappingRules(c.namespace).
		UpdateStatus(c.ctx, fp, metav1.UpdateOptions{})
	return err
}

func (c *MappingRuleController) onAdd(obj interface{}) {
	f := obj.(*v1alpha1.MappingRule)
	f = f.DeepCopy()
	c.logger.Info("onAdd",
		"MappingRule", objref.KObj(f),
	)

	c.mut.Lock()
	defer c.mut.Unlock()

	c.cache[f.Name] = f

	c.syncFunc()
}

func (c *MappingRuleController) onUpdate(oldObj, newObj interface{}) {
	f := newObj.(*v1alpha1.MappingRule)
	f = f.DeepCopy()
	c.logger.Info("onUpdate",
		"MappingRule", objref.KObj(f),
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

func (c *MappingRuleController) onDelete(obj interface{}) {
	f := obj.(*v1alpha1.MappingRule)
	c.logger.Info("onDelete",
		"MappingRule", objref.KObj(f),
	)

	c.mut.Lock()
	defer c.mut.Unlock()

	delete(c.cache, f.Name)

	c.syncFunc()
}

func (c *MappingRuleController) Apply(ctx context.Context, rule *v1alpha1.MappingRule) (err error) {
	logger := logr.FromContextOrDiscard(ctx)
	ori, err := c.clientset.
		FerryV1alpha1().
		MappingRules(rule.Namespace).
		Get(ctx, rule.Name, metav1.GetOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			return fmt.Errorf("get mapping rule %s: %w", objref.KObj(rule), err)
		}
		logger.Info("Creating Service", "Service", objref.KObj(rule))
		_, err = c.clientset.
			FerryV1alpha1().
			MappingRules(rule.Namespace).
			Create(ctx, rule, metav1.CreateOptions{
				FieldManager: consts.LabelFerryManagedByValue,
			})
		if err != nil {
			return fmt.Errorf("create mapping rule %s: %w", objref.KObj(rule), err)
		}
	} else {
		_, err = c.clientset.
			FerryV1alpha1().
			MappingRules(rule.Namespace).
			Update(ctx, ori, metav1.UpdateOptions{
				FieldManager: consts.LabelFerryManagedByValue,
			})
		if err != nil {
			return fmt.Errorf("update mapping rule %s: %w", objref.KObj(rule), err)
		}
	}
	return nil
}

func (c *MappingRuleController) Delete(ctx context.Context, rule *v1alpha1.MappingRule) (err error) {
	logger := logr.FromContextOrDiscard(ctx)
	logger.Info("Deleting Service", "Service", objref.KObj(rule))

	err = c.clientset.
		FerryV1alpha1().
		MappingRules(rule.Namespace).
		Delete(ctx, rule.Name, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("delete mapping rule  %s: %w", objref.KObj(rule), err)
	}
	return nil
}

func (c *MappingRuleController) Sync(ctx context.Context) {
	mappingRules := c.List()

	newerMappingRules := GroupMappingRules(mappingRules)
	defer func() {
		c.cacheMappingRules = newerMappingRules
	}()
	logger := c.logger.WithName("sync")

	updated, deleted := CalculateMappingRulesPatch(c.cacheMappingRules, newerMappingRules)

	cis := CalculateClusterInformationStatus(updated)
	for name, status := range cis {
		err := c.clusterInformationController.UpdateStatus(name, status.ImportedFrom, status.ExportedTo, "Working")
		if err != nil {
			logger.Error(err, "update cluster information status")
		}
	}

	for _, r := range deleted {
		logger := logger.WithValues("export", r.Export, "import", r.Import)
		logger.Info("Delete data plane")
		c.cleanupDataPlane(r.Export, r.Import)
	}

	for _, r := range updated {
		logger := logger.WithValues("export", r.Export, "import", r.Import)
		logger.Info("Update data plane")
		dataPlane, err := c.startDataPlane(ctx, r.Export, r.Import)
		if err != nil {
			logger.Error(err, "start data plane")
			continue
		}
		if newerMappingRules[r.Export] != nil && newerMappingRules[r.Export][r.Import] != nil {
			older := c.cacheMappingRules[r.Export][r.Import]
			newer := newerMappingRules[r.Export][r.Import]
			updated, deleted := utils.CalculatePatchResources(older, newer)
			for _, rule := range deleted {
				logger.Info("Delete rule", "rule", rule)
				dataPlane.Unregistry(
					objref.ObjectRef{Name: rule.Spec.Export.Service.Name, Namespace: rule.Spec.Export.Service.Namespace},
					objref.ObjectRef{Name: rule.Spec.Import.Service.Name, Namespace: rule.Spec.Import.Service.Namespace},
				)
			}

			for _, rule := range updated {
				logger.Info("Update rule", "rule", rule)
				dataPlane.Registry(
					objref.ObjectRef{Name: rule.Spec.Export.Service.Name, Namespace: rule.Spec.Export.Service.Namespace},
					objref.ObjectRef{Name: rule.Spec.Import.Service.Name, Namespace: rule.Spec.Import.Service.Namespace},
				)
				err = c.UpdateStatus(rule.Name, "Worked")
				if err != nil {
					logger.Error(err, "failed to update status")
				}
			}
		}
		dataPlane.try.Try()
	}
	for name, status := range cis {
		err := c.clusterInformationController.UpdateStatus(name, status.ImportedFrom, status.ExportedTo, "Worked")
		if err != nil {
			logger.Error(err, "update cluster information status")
		}
	}
	return
}

func (c *MappingRuleController) cleanupDataPlane(exportClusterName, importClusterName string) {
	key := ClusterPair{
		Export: exportClusterName,
		Import: importClusterName,
	}
	dataPlane := c.cacheDataPlaneController[key]
	if dataPlane != nil {
		dataPlane.Close()
		delete(c.cacheDataPlaneController, key)
	}
}

func (c *MappingRuleController) startDataPlane(ctx context.Context, exportClusterName, importClusterName string) (*dataPlaneController, error) {
	key := ClusterPair{
		Export: exportClusterName,
		Import: importClusterName,
	}
	dataPlane := c.cacheDataPlaneController[key]
	if dataPlane != nil {
		return dataPlane, nil
	}

	exportClientset := c.clusterInformationController.Clientset(exportClusterName)
	if exportClientset == nil {
		return nil, fmt.Errorf("not found clientset %q", exportClusterName)
	}
	importClientset := c.clusterInformationController.Clientset(importClusterName)
	if importClientset == nil {
		return nil, fmt.Errorf("not found clientset %q", importClusterName)
	}

	exportCluster := c.clusterInformationController.Get(exportClusterName)
	if exportCluster == nil {
		return nil, fmt.Errorf("not found cluster information %q", exportClusterName)
	}

	importCluster := c.clusterInformationController.Get(importClusterName)
	if importCluster == nil {
		return nil, fmt.Errorf("not found cluster information %q", importClusterName)
	}

	dataPlane = newDataPlaneController(dataPlaneControllerConfig{
		ClusterInformationController: c.clusterInformationController,
		ImportClusterName:            importClusterName,
		ExportClusterName:            exportClusterName,
		ExportCluster:                exportCluster,
		ImportCluster:                importCluster,
		ExportClientset:              exportClientset,
		ImportClientset:              importClientset,
		Logger:                       c.logger.WithName("data-plane").WithName(importClusterName).WithValues("export", exportClusterName, "import", importClusterName),
		SourceResourceBuilder:        router.ResourceBuilders{original.IngressBuilder},
		DestinationResourceBuilder:   router.ResourceBuilders{original.EgressBuilder, original.ServiceEgressDiscoveryBuilder},
	})
	c.cacheDataPlaneController[key] = dataPlane

	err := dataPlane.Start(ctx)
	if err != nil {
		return nil, err
	}
	return dataPlane, nil
}

type clusterStatus struct {
	ExportedTo   []string
	ImportedFrom []string
}

func CalculateClusterInformationStatus(updated []ClusterPair) map[string]clusterStatus {
	out := map[string]clusterStatus{}
	for _, u := range updated {
		imported := out[u.Import]
		imported.ImportedFrom = append(imported.ImportedFrom, u.Export)
		out[u.Import] = imported

		exported := out[u.Export]
		exported.ExportedTo = append(exported.ExportedTo, u.Import)
		out[u.Export] = exported
	}
	return out
}

func GroupMappingRules(rules []*v1alpha1.MappingRule) map[string]map[string][]*v1alpha1.MappingRule {
	mapping := map[string]map[string][]*v1alpha1.MappingRule{}

	for _, spec := range rules {
		rule := spec.Spec
		export := rule.Export
		impor := rule.Import

		if export.ClusterName == "" || impor.ClusterName == "" || impor.ClusterName == export.ClusterName {
			continue
		}

		if _, ok := mapping[export.ClusterName]; !ok {
			mapping[export.ClusterName] = map[string][]*v1alpha1.MappingRule{}
		}

		if _, ok := mapping[export.ClusterName][impor.ClusterName]; !ok {
			mapping[export.ClusterName][impor.ClusterName] = []*v1alpha1.MappingRule{}
		}

		mapping[export.ClusterName][impor.ClusterName] = append(mapping[export.ClusterName][impor.ClusterName], spec)
	}
	return mapping
}

type ClusterPair struct {
	Export string
	Import string
}

func CalculateMappingRulesPatch(older, newer map[string]map[string][]*v1alpha1.MappingRule) (updated, deleted []ClusterPair) {
	exist := map[ClusterPair]struct{}{}

	for exportName, other := range older {
		for importName := range other {
			r := ClusterPair{
				Export: exportName,
				Import: importName,
			}
			exist[r] = struct{}{}
		}
	}

	for exportName, other := range newer {
		for importName := range other {
			r := ClusterPair{
				Export: exportName,
				Import: importName,
			}
			updated = append(updated, r)
			delete(exist, r)
		}
	}

	for r := range exist {
		deleted = append(deleted, r)
	}
	return updated, deleted
}
