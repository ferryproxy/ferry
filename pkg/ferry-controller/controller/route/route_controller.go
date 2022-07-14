package route

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	"github.com/ferryproxy/api/apis/traffic/v1alpha2"
	versioned "github.com/ferryproxy/client-go/generated/clientset/versioned"
	externalversions "github.com/ferryproxy/client-go/generated/informers/externalversions"
	"github.com/ferryproxy/ferry/pkg/consts"
	"github.com/ferryproxy/ferry/pkg/ferry-controller/controller/mapping"
	"github.com/ferryproxy/ferry/pkg/utils/objref"
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

type RouteControllerConfig struct {
	Logger       logr.Logger
	Config       *restclient.Config
	ClusterCache mapping.ClusterCache
	Namespace    string
	SyncFunc     func()
}

type RouteController struct {
	ctx                    context.Context
	mut                    sync.RWMutex
	config                 *restclient.Config
	clientset              *versioned.Clientset
	clusterCache           mapping.ClusterCache
	cache                  map[string]*v1alpha2.Route
	cacheMappingController map[clusterPair]*mapping.MappingController
	cacheRoutes            map[clusterPair][]*v1alpha2.Route
	namespace              string
	syncFunc               func()
	logger                 logr.Logger
}

func NewRouteController(conf *RouteControllerConfig) *RouteController {
	return &RouteController{
		config:                 conf.Config,
		namespace:              conf.Namespace,
		clusterCache:           conf.ClusterCache,
		logger:                 conf.Logger,
		syncFunc:               conf.SyncFunc,
		cache:                  map[string]*v1alpha2.Route{},
		cacheMappingController: map[clusterPair]*mapping.MappingController{},
		cacheRoutes:            map[clusterPair][]*v1alpha2.Route{},
	}
}

func (c *RouteController) list() []*v1alpha2.Route {
	var list []*v1alpha2.Route
	for _, v := range c.cache {
		item := c.cache[v.Name]
		if item == nil {
			continue
		}
		list = append(list, item)
	}
	return list
}

func (c *RouteController) Run(ctx context.Context) error {
	c.logger.Info("Route controller started")
	defer c.logger.Info("Route controller stopped")

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
		Routes().
		Informer()
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.onAdd,
		UpdateFunc: c.onUpdate,
		DeleteFunc: c.onDelete,
	})

	informer.Run(ctx.Done())
	return nil
}

func (c *RouteController) updateStatus(name string, phase string) error {
	fp := c.cache[name]
	if fp == nil {
		return fmt.Errorf("not found Route %s", name)
	}

	fp = fp.DeepCopy()

	fp.Status.LastSynchronizationTimestamp = metav1.Now()
	fp.Status.Import = fmt.Sprintf("%s.%s/%s", fp.Spec.Import.Service.Name, fp.Spec.Import.Service.Namespace, fp.Spec.Import.HubName)
	fp.Status.Export = fmt.Sprintf("%s.%s/%s", fp.Spec.Export.Service.Name, fp.Spec.Export.Service.Namespace, fp.Spec.Export.HubName)
	fp.Status.Phase = phase
	_, err := c.clientset.
		TrafficV1alpha2().
		Routes(fp.Namespace).
		UpdateStatus(c.ctx, fp, metav1.UpdateOptions{})
	return err
}

func (c *RouteController) onAdd(obj interface{}) {
	f := obj.(*v1alpha2.Route)
	f = f.DeepCopy()
	c.logger.Info("onAdd",
		"route", objref.KObj(f),
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

func (c *RouteController) onUpdate(oldObj, newObj interface{}) {
	f := newObj.(*v1alpha2.Route)
	f = f.DeepCopy()
	c.logger.Info("onUpdate",
		"route", objref.KObj(f),
	)

	c.mut.Lock()
	defer c.mut.Unlock()

	if reflect.DeepEqual(c.cache[f.Name].Spec, f.Spec) {
		c.cache[f.Name] = f
		return
	}

	c.cache[f.Name] = f

	c.syncFunc()

	err := c.updateStatus(f.Name, "Pending")
	if err != nil {
		c.logger.Error(err, "failed to update status")
	}
}

func (c *RouteController) onDelete(obj interface{}) {
	f := obj.(*v1alpha2.Route)
	c.logger.Info("onDelete",
		"route", objref.KObj(f),
	)

	c.mut.Lock()
	defer c.mut.Unlock()

	delete(c.cache, f.Name)

	c.syncFunc()
}

func (c *RouteController) Sync(ctx context.Context) {
	c.mut.RLock()
	defer c.mut.RUnlock()

	routes := c.list()

	newerRoutes := groupRoutes(routes)
	defer func() {
		c.cacheRoutes = newerRoutes
	}()
	logger := c.logger.WithName("sync")

	updated, deleted := calculateRoutesPatch(c.cacheRoutes, newerRoutes)

	for _, key := range deleted {
		logger := logger.WithValues("export", key.Export, "import", key.Import)
		logger.Info("Delete mapping controller")
		c.cleanupMappingController(key)
	}

	for _, key := range updated {
		logger := logger.WithValues("export", key.Export, "import", key.Import)
		logger.Info("Update mapping controller")
		mc, err := c.startMappingController(ctx, key)
		if err != nil {
			logger.Error(err, "start mapping controller")
			continue
		}

		mc.SetRoutes(newerRoutes[key])

		mc.Sync()

		for _, rule := range newerRoutes[key] {
			err := c.updateStatus(rule.Name, "Worked")
			if err != nil {
				c.logger.Error(err, "failed to update status")
			}
		}
	}
	return
}

func (c *RouteController) cleanupMappingController(key clusterPair) {
	mc := c.cacheMappingController[key]
	if mc != nil {
		mc.Close()
		delete(c.cacheMappingController, key)
	}
}

func (c *RouteController) startMappingController(ctx context.Context, key clusterPair) (*mapping.MappingController, error) {
	mc := c.cacheMappingController[key]
	if mc != nil {
		return mc, nil
	}

	exportClientset := c.clusterCache.Clientset(key.Export)
	if exportClientset == nil {
		return nil, fmt.Errorf("not found clientset %q", key.Export)
	}
	importClientset := c.clusterCache.Clientset(key.Import)
	if importClientset == nil {
		return nil, fmt.Errorf("not found clientset %q", key.Import)
	}

	exportCluster := c.clusterCache.GetHub(key.Export)
	if exportCluster == nil {
		return nil, fmt.Errorf("not found cluster information %q", key.Export)
	}

	importCluster := c.clusterCache.GetHub(key.Import)
	if importCluster == nil {
		return nil, fmt.Errorf("not found cluster information %q", key.Import)
	}

	mc = mapping.NewMappingController(mapping.MappingControllerConfig{
		Namespace:     consts.FerryTunnelNamespace,
		ClusterCache:  c.clusterCache,
		ImportHubName: key.Import,
		ExportHubName: key.Export,
		Logger: c.logger.WithName("data-plane").
			WithName(key.Import).
			WithValues("export", key.Export, "import", key.Import),
	})
	c.cacheMappingController[key] = mc

	err := mc.Start(ctx)
	if err != nil {
		return nil, err
	}
	return mc, nil
}

func groupRoutes(rules []*v1alpha2.Route) map[clusterPair][]*v1alpha2.Route {
	mapping := map[clusterPair][]*v1alpha2.Route{}

	for _, spec := range rules {
		rule := spec.Spec
		export := rule.Export
		impor := rule.Import

		if export.HubName == "" || impor.HubName == "" || impor.HubName == export.HubName {
			continue
		}

		key := clusterPair{
			Export: rule.Export.HubName,
			Import: rule.Import.HubName,
		}

		if _, ok := mapping[key]; !ok {
			mapping[key] = []*v1alpha2.Route{}
		}

		mapping[key] = append(mapping[key], spec)
	}
	return mapping
}

type clusterPair struct {
	Export string
	Import string
}

func calculateRoutesPatch(older, newer map[clusterPair][]*v1alpha2.Route) (updated, deleted []clusterPair) {
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
