package mapping

import (
	"context"
	"sync"
	"time"

	"github.com/ferryproxy/api/apis/traffic/v1alpha2"
	"github.com/ferryproxy/ferry/pkg/consts"
	"github.com/ferryproxy/ferry/pkg/ferry-controller/router"
	"github.com/ferryproxy/ferry/pkg/ferry-controller/router/resource"
	"github.com/ferryproxy/ferry/pkg/utils/diffobjs"
	"github.com/ferryproxy/ferry/pkg/utils/trybuffer"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

type ClusterCache interface {
	ListServices(name string) []*corev1.Service
	GetHub(name string) *v1alpha2.Hub
	GetHubGateway(hubName string, forHub string) v1alpha2.HubSpecGateway
	GetIdentity(name string) string
	Clientset(name string) kubernetes.Interface
	LoadPortPeer(importHubName string, list *corev1.ServiceList)
	GetPortPeer(importHubName string, cluster, namespace, name string, port int32) int32
	RegistryServiceCallback(exportHubName, importHubName string, cb func())
	UnregistryServiceCallback(exportHubName, importHubName string)
}

type MappingControllerConfig struct {
	Namespace     string
	ExportHubName string
	ImportHubName string
	ClusterCache  ClusterCache
	Logger        logr.Logger
}

func NewMappingController(conf MappingControllerConfig) *MappingController {
	return &MappingController{
		namespace:      conf.Namespace,
		importHubName:  conf.ImportHubName,
		exportHubName:  conf.ExportHubName,
		logger:         conf.Logger,
		clusterCache:   conf.ClusterCache,
		cacheResources: map[string][]resource.Resourcer{},
	}
}

type MappingController struct {
	mut sync.Mutex
	ctx context.Context

	namespace string
	labels    map[string]string

	exportHubName string
	importHubName string

	router       *router.Router
	solution     *router.Solution
	clusterCache ClusterCache

	cacheResources map[string][]resource.Resourcer
	logger         logr.Logger
	way            []string

	try *trybuffer.TryBuffer

	isClose bool
}

func (d *MappingController) Start(ctx context.Context) error {
	d.logger.Info("DataPlane controller started")
	defer func() {
		d.logger.Info("DataPlane controller stopped")
	}()
	d.ctx = ctx

	d.solution = router.NewSolution(router.SolutionConfig{
		GetHubGateway: d.clusterCache.GetHubGateway,
	})

	// Mark managed by ferry
	opt := metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(d.getLabel()).String(),
	}

	way, err := d.solution.CalculateWays(d.exportHubName, d.importHubName)
	if err != nil {
		d.logger.Error(err, "calculate ways")
		return err
	}
	d.way = way

	for _, w := range way {
		err := d.loadLastConfigMap(ctx, w, opt)
		if err != nil {
			return err
		}
	}

	err = d.loadLastService(ctx, way[len(way)-1], opt)
	if err != nil {
		return err
	}

	d.try = trybuffer.NewTryBuffer(d.sync, time.Second/2)

	d.clusterCache.RegistryServiceCallback(d.exportHubName, d.importHubName, d.Sync)

	d.router = router.NewRouter(router.RouterConfig{
		Labels:        d.getLabel(),
		Namespace:     d.namespace,
		ExportHubName: d.exportHubName,
		ImportHubName: d.importHubName,
		GetIdentity:   d.clusterCache.GetIdentity,
		ListServices:  d.clusterCache.ListServices,
		GetHubGateway: d.clusterCache.GetHubGateway,
		GetPortPeer:   d.clusterCache.GetPortPeer,
	})

	return nil
}

func (d *MappingController) Way() []string {
	d.mut.Lock()
	defer d.mut.Unlock()
	return d.way
}

func (d *MappingController) Sync() {
	d.try.Try()
}

func (d *MappingController) SetRoutes(rules []*v1alpha2.Route) {
	d.mut.Lock()
	defer d.mut.Unlock()
	d.router.SetRoutes(rules)
}

func (d *MappingController) loadLastConfigMap(ctx context.Context, name string, opt metav1.ListOptions) error {
	cmList, err := d.clusterCache.Clientset(name).
		CoreV1().
		ConfigMaps(d.namespace).
		List(ctx, opt)
	if err != nil {
		return err
	}
	for _, item := range cmList.Items {
		d.cacheResources[name] = append(d.cacheResources[name], resource.ConfigMap{item.DeepCopy()})
	}
	return nil
}

func (d *MappingController) loadLastService(ctx context.Context, name string, opt metav1.ListOptions) error {
	svcList, err := d.clusterCache.Clientset(name).
		CoreV1().
		Services("").
		List(ctx, opt)
	if err != nil {
		return err
	}
	for _, item := range svcList.Items {
		d.cacheResources[name] = append(d.cacheResources[name], resource.Service{item.DeepCopy()})
	}

	d.clusterCache.LoadPortPeer(name, svcList)
	return nil
}

func (d *MappingController) getLabel() map[string]string {
	if d.labels != nil {
		return d.labels
	}
	d.labels = map[string]string{
		consts.LabelFerryManagedByKey:    consts.LabelFerryManagedByValue,
		consts.LabelFerryExportedFromKey: d.exportHubName,
		consts.LabelFerryImportedToKey:   d.importHubName,
	}
	return d.labels
}

func (d *MappingController) sync() {
	d.mut.Lock()
	defer d.mut.Unlock()

	if d.isClose {
		return
	}
	ctx := d.ctx

	way, err := d.solution.CalculateWays(d.exportHubName, d.importHubName)
	if err != nil {
		d.logger.Error(err, "calculate ways")
		return
	}
	d.way = way

	resources, err := d.router.BuildResource(way)
	if err != nil {
		d.logger.Error(err, "build resource")
		return
	}
	defer func() {
		d.cacheResources = resources
	}()

	for hubName, updated := range resources {
		cacheResource := d.cacheResources[hubName]
		deleled := diffobjs.ShouldDeleted(cacheResource, updated)
		cli := d.clusterCache.Clientset(hubName)
		for _, r := range updated {
			err := r.Apply(ctx, cli)
			if err != nil {
				d.logger.Error(err, "Apply resource", "hub", hubName)
			}
		}

		for _, r := range deleled {
			err := r.Delete(ctx, cli)
			if err != nil {
				d.logger.Error(err, "Delete resource", "hub", hubName)
			}
		}
	}

	for hubName, caches := range d.cacheResources {
		v, ok := resources[hubName]
		if ok && len(v) != 0 {
			continue
		}
		cli := d.clusterCache.Clientset(hubName)
		for _, r := range caches {
			err := r.Delete(ctx, cli)
			if err != nil {
				d.logger.Error(err, "Delete resource", "hub", hubName)
			}
		}
	}

	return
}

func (d *MappingController) Close() {
	d.mut.Lock()
	defer d.mut.Unlock()

	if d.isClose {
		return
	}
	d.isClose = true
	d.clusterCache.UnregistryServiceCallback(d.exportHubName, d.importHubName)
	d.try.Close()

	ctx := context.Background()

	for hubName, caches := range d.cacheResources {
		cli := d.clusterCache.Clientset(hubName)
		for _, r := range caches {
			err := r.Delete(ctx, cli)
			if err != nil {
				d.logger.Error(err, "Delete resource", "hub", hubName)
			}
		}
	}
}
