package mapping_rule

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	"github.com/ferry-proxy/api/apis/ferry/v1alpha1"
	versioned "github.com/ferry-proxy/client-go/generated/clientset/versioned"
	externalversions "github.com/ferry-proxy/client-go/generated/informers/externalversions"
	"github.com/ferry-proxy/ferry/pkg/ferry-controller/router"
	original "github.com/ferry-proxy/ferry/pkg/ferry-controller/router/tunnel"
	"github.com/ferry-proxy/ferry/pkg/utils/objref"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

type ClusterCache interface {
	ListServices(name string) []*corev1.Service
	GetClusterInformation(name string) *v1alpha1.ClusterInformation
	GetIdentity(name string) string
	Clientset(name string) *kubernetes.Clientset
	LoadPortPeer(importClusterName string, list *corev1.ServiceList)
	GetPortPeer(importClusterName string, cluster, namespace, name string, port int32) int32
	RegistryServiceCallback(exportClusterName, importClusterName string, cb func())
	UnregistryServiceCallback(exportClusterName, importClusterName string)
}

type MappingRuleControllerConfig struct {
	Logger       logr.Logger
	Config       *restclient.Config
	ClusterCache ClusterCache
	Namespace    string
	SyncFunc     func()
}

type MappingRuleController struct {
	ctx                      context.Context
	mut                      sync.RWMutex
	config                   *restclient.Config
	clientset                *versioned.Clientset
	clusterCache             ClusterCache
	cache                    map[string]*v1alpha1.MappingRule
	cacheDataPlaneController map[clusterPair]*dataPlaneController
	cacheMappingRules        map[clusterPair][]*v1alpha1.MappingRule
	namespace                string
	syncFunc                 func()
	logger                   logr.Logger
}

func NewMappingRuleController(conf *MappingRuleControllerConfig) *MappingRuleController {
	return &MappingRuleController{
		config:                   conf.Config,
		namespace:                conf.Namespace,
		clusterCache:             conf.ClusterCache,
		logger:                   conf.Logger,
		syncFunc:                 conf.SyncFunc,
		cache:                    map[string]*v1alpha1.MappingRule{},
		cacheDataPlaneController: map[clusterPair]*dataPlaneController{},
		cacheMappingRules:        map[clusterPair][]*v1alpha1.MappingRule{},
	}
}

func (c *MappingRuleController) list() []*v1alpha1.MappingRule {
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

func (c *MappingRuleController) updateStatus(name string, phase string) error {
	fp := c.cache[name]
	if fp == nil {
		return fmt.Errorf("not found MappingRule %s", name)
	}

	fp = fp.DeepCopy()

	fp.Status.LastSynchronizationTimestamp = metav1.Now()
	fp.Status.Import = fmt.Sprintf("%s.%s.svc.%s", fp.Spec.Import.Service.Name, fp.Spec.Import.Service.Namespace, fp.Spec.Import.ClusterName)
	fp.Status.Export = fmt.Sprintf("%s.%s.svc.%s", fp.Spec.Export.Service.Name, fp.Spec.Export.Service.Namespace, fp.Spec.Export.ClusterName)
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

	err := c.updateStatus(f.Name, "Pending")
	if err != nil {
		c.logger.Error(err, "failed to update status")
	}
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

	err := c.updateStatus(f.Name, "Pending")
	if err != nil {
		c.logger.Error(err, "failed to update status")
	}
}

func (c *MappingRuleController) Sync(ctx context.Context) {
	c.mut.RLock()
	defer c.mut.RUnlock()

	mappingRules := c.list()

	for _, rule := range mappingRules {
		err := c.updateStatus(rule.Name, "Working")
		if err != nil {
			c.logger.Error(err, "failed to update status")
		}
	}

	newerMappingRules := groupMappingRules(mappingRules)
	defer func() {
		c.cacheMappingRules = newerMappingRules
	}()
	logger := c.logger.WithName("sync")

	updated, deleted := calculateMappingRulesPatch(c.cacheMappingRules, newerMappingRules)

	for _, key := range deleted {
		logger := logger.WithValues("export", key.Export, "import", key.Import)
		logger.Info("Delete data plane")
		c.cleanupDataPlane(key)
	}

	for _, key := range updated {
		logger := logger.WithValues("export", key.Export, "import", key.Import)
		logger.Info("Update data plane")
		dataPlane, err := c.startDataPlane(ctx, key)
		if err != nil {
			logger.Error(err, "start data plane")
			continue
		}

		dataPlane.SetMappingRules(newerMappingRules[key])

		dataPlane.Sync()

		for _, rule := range newerMappingRules[key] {
			err := c.updateStatus(rule.Name, "Worked")
			if err != nil {
				c.logger.Error(err, "failed to update status")
			}
		}
	}
	return
}

func (c *MappingRuleController) cleanupDataPlane(key clusterPair) {
	dataPlane := c.cacheDataPlaneController[key]
	if dataPlane != nil {
		dataPlane.Close()
		delete(c.cacheDataPlaneController, key)
	}
}

func (c *MappingRuleController) startDataPlane(ctx context.Context, key clusterPair) (*dataPlaneController, error) {
	dataPlane := c.cacheDataPlaneController[key]
	if dataPlane != nil {
		return dataPlane, nil
	}

	exportClientset := c.clusterCache.Clientset(key.Export)
	if exportClientset == nil {
		return nil, fmt.Errorf("not found clientset %q", key.Export)
	}
	importClientset := c.clusterCache.Clientset(key.Import)
	if importClientset == nil {
		return nil, fmt.Errorf("not found clientset %q", key.Import)
	}

	exportCluster := c.clusterCache.GetClusterInformation(key.Export)
	if exportCluster == nil {
		return nil, fmt.Errorf("not found cluster information %q", key.Export)
	}

	importCluster := c.clusterCache.GetClusterInformation(key.Import)
	if importCluster == nil {
		return nil, fmt.Errorf("not found cluster information %q", key.Import)
	}

	dataPlane = newDataPlaneController(dataPlaneControllerConfig{
		ClusterCache:               c.clusterCache,
		ImportClusterName:          key.Import,
		ExportClusterName:          key.Export,
		ExportClientset:            exportClientset,
		ImportClientset:            importClientset,
		SourceResourceBuilder:      router.ResourceBuilders{original.IngressBuilder},
		DestinationResourceBuilder: router.ResourceBuilders{original.EgressBuilder, original.ServiceEgressDiscoveryBuilder},
		Logger: c.logger.WithName("data-plane").
			WithName(key.Import).
			WithValues("export", key.Export, "import", key.Import),
	})
	c.cacheDataPlaneController[key] = dataPlane

	err := dataPlane.Start(ctx)
	if err != nil {
		return nil, err
	}
	return dataPlane, nil
}

func groupMappingRules(rules []*v1alpha1.MappingRule) map[clusterPair][]*v1alpha1.MappingRule {
	mapping := map[clusterPair][]*v1alpha1.MappingRule{}

	for _, spec := range rules {
		rule := spec.Spec
		export := rule.Export
		impor := rule.Import

		if export.ClusterName == "" || impor.ClusterName == "" || impor.ClusterName == export.ClusterName {
			continue
		}

		key := clusterPair{
			Export: rule.Export.ClusterName,
			Import: rule.Import.ClusterName,
		}

		if _, ok := mapping[key]; !ok {
			mapping[key] = []*v1alpha1.MappingRule{}
		}

		mapping[key] = append(mapping[key], spec)
	}
	return mapping
}

type clusterPair struct {
	Export string
	Import string
}

func calculateMappingRulesPatch(older, newer map[clusterPair][]*v1alpha1.MappingRule) (updated, deleted []clusterPair) {
	exist := map[clusterPair]struct{}{}

	for key := range older {
		exist[key] = struct{}{}
	}

	for key := range newer {
		updated = append(updated, key)
		delete(exist, key)
	}

	for r := range exist {
		deleted = append(deleted, r)
	}
	return updated, deleted
}
