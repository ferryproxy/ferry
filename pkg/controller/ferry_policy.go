package controller

import (
	"context"
	"fmt"
	"sync"

	"github.com/ferry-proxy/api/apis/ferry/v1alpha1"
	versioned "github.com/ferry-proxy/client-go/generated/clientset/versioned"
	externalversions "github.com/ferry-proxy/client-go/generated/informers/externalversions"
	"github.com/ferry-proxy/ferry/pkg/utils"
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
)

type ferryPolicyControllerConfig struct {
	Logger    logr.Logger
	Config    *restclient.Config
	Namespace string
	SyncFunc  func(context.Context, *v1alpha1.FerryPolicy)
}

type ferryPolicyController struct {
	ctx       context.Context
	mut       sync.RWMutex
	config    *restclient.Config
	cache     map[string]*v1alpha1.FerryPolicy
	mapCancel map[string]func()
	namespace string
	syncFunc  func(context.Context, *v1alpha1.FerryPolicy)
	logger    logr.Logger
}

func newFerryPolicyController(conf *ferryPolicyControllerConfig) *ferryPolicyController {
	return &ferryPolicyController{
		config:    conf.Config,
		namespace: conf.Namespace,
		logger:    conf.Logger,
		syncFunc:  conf.SyncFunc,
		cache:     map[string]*v1alpha1.FerryPolicy{},
		mapCancel: map[string]func(){},
	}
}

func (c *ferryPolicyController) List() []*v1alpha1.FerryPolicy {
	c.mut.RLock()
	defer c.mut.RUnlock()
	var list []*v1alpha1.FerryPolicy
	for _, v := range c.cache {
		item := c.cache[v.Name]
		if item == nil {
			continue
		}
		list = append(list, item)
	}
	return list
}

func (c *ferryPolicyController) Get(name string) *v1alpha1.FerryPolicy {
	c.mut.RLock()
	defer c.mut.RUnlock()
	return c.cache[name]
}

func (c *ferryPolicyController) Run(ctx context.Context) error {
	c.logger.Info("FerryPolicy controller started")
	defer c.logger.Info("FerryPolicy controller stopped")

	clientset, err := versioned.NewForConfig(c.config)
	if err != nil {
		return err
	}
	c.ctx = ctx
	informerFactory := externalversions.NewSharedInformerFactoryWithOptions(clientset, 0,
		externalversions.WithNamespace(c.namespace))
	informer := informerFactory.Ferry().
		V1alpha1().
		FerryPolicies().
		Informer()
	informer.AddEventHandler(c)
	informer.Run(ctx.Done())
	return nil
}

func (c *ferryPolicyController) OnAdd(obj interface{}) {
	f := obj.(*v1alpha1.FerryPolicy)
	f = f.DeepCopy()
	c.logger.Info("OnAdd",
		"FerryPolicy", utils.KObj(f),
	)

	c.mut.Lock()
	defer c.mut.Unlock()

	c.cache[f.Name] = f

	ctx, cancel := context.WithCancel(c.ctx)
	c.mapCancel[f.Name] = cancel
	c.syncFunc(ctx, f)
}

func (c *ferryPolicyController) OnUpdate(oldObj, newObj interface{}) {
	f := newObj.(*v1alpha1.FerryPolicy)
	f = f.DeepCopy()
	c.logger.Info("OnUpdate",
		"FerryPolicy", utils.KObj(f),
	)

	c.mut.Lock()
	defer c.mut.Unlock()

	c.cache[f.Name] = f

	cancel, ok := c.mapCancel[f.Name]
	if ok && cancel != nil {
		cancel()
	}

	ctx, cancel := context.WithCancel(c.ctx)
	c.mapCancel[f.Name] = cancel
	c.syncFunc(ctx, f)
}

func (c *ferryPolicyController) OnDelete(obj interface{}) {
	f := obj.(*v1alpha1.FerryPolicy)
	c.logger.Info("OnDelete",
		"FerryPolicy", utils.KObj(f),
	)

	c.mut.Lock()
	defer c.mut.Unlock()

	delete(c.cache, f.Name)

	cancel, ok := c.mapCancel[f.Name]
	if ok && cancel != nil {
		cancel()
	}

	delete(c.mapCancel, f.Name)
	c.syncFunc(context.Background(), f)
}

func getPort(ctx context.Context, clientset *kubernetes.Clientset, route *v1alpha1.ClusterInformationSpecRoute) (int32, error) {
	if route == nil {
		return 31087, nil
	}
	if route.Port != 0 {
		return route.Port, nil
	}
	if route.ServiceNamespace == "" && route.ServiceName == "" {
		return 31087, nil
	}
	if route.ServiceNamespace == "" {
		return 0, fmt.Errorf("ServiceNamespace is empty")
	}
	if route.ServiceName == "" {
		return 0, fmt.Errorf("ServiceName is empty")
	}
	ep, err := clientset.CoreV1().Endpoints(route.ServiceNamespace).Get(ctx, route.ServiceName, metav1.GetOptions{})
	if err != nil {
		return 0, err
	}
	if len(ep.Subsets) == 0 {
		return 0, fmt.Errorf("Endpoints's Subsets is empty")
	}

	if len(ep.Subsets[0].Ports) == 0 {
		return 0, fmt.Errorf("Endpoints's Subsets[0].Ports is empty")
	}

	for _, port := range ep.Subsets[0].Ports {
		if port.Port != 0 {
			return port.Port, nil
		}
	}
	return 31087, nil
}

func getIPs(ctx context.Context, clientset *kubernetes.Clientset, route *v1alpha1.ClusterInformationSpecRoute) ([]string, error) {
	if route == nil {
		return nil, nil
	}
	if route.IP != "" {
		return []string{route.IP}, nil
	}
	if route.ServiceNamespace == "" && route.ServiceName == "" {
		return nil, nil
	}
	if route.ServiceNamespace == "" {
		return nil, fmt.Errorf("ServiceNamespace is empty")
	}
	if route.ServiceName == "" {
		return nil, fmt.Errorf("ServiceName is empty")
	}
	ep, err := clientset.CoreV1().Endpoints(route.ServiceNamespace).Get(ctx, route.ServiceName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	if len(ep.Subsets) == 0 {
		return nil, fmt.Errorf("Endpoints's Subsets is empty")
	}

	if len(ep.Subsets[0].Addresses) == 0 {
		return nil, fmt.Errorf("Endpoints's Subsets[0].Addresses is empty")
	}

	ips := []string{}
	for _, address := range ep.Subsets[0].Addresses {
		if address.IP == "" {
			continue
		}
		ips = append(ips, address.IP)
	}
	return ips, nil
}
