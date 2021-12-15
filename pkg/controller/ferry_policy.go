package controller

import (
	"context"
	"fmt"
	"sync"

	"github.com/ferry-proxy/ferry/api/v1alpha1"
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/cache"
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

func (c *ferryPolicyController) List() []string {
	c.mut.RLock()
	defer c.mut.RUnlock()
	var list []string
	for _, v := range c.cache {
		list = append(list, v.Name)
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
	cache, err := cache.New(c.config, cache.Options{
		Namespace: c.namespace,
	})
	if err != nil {
		return err
	}
	informer, err := cache.GetInformer(ctx, &v1alpha1.FerryPolicy{})
	if err != nil {
		return err
	}
	informer.AddEventHandler(c)
	c.ctx = ctx
	return cache.Start(ctx)
}

func (c *ferryPolicyController) OnAdd(obj interface{}) {
	f := obj.(*v1alpha1.FerryPolicy)
	f = f.DeepCopy()
	c.logger.Info("OnAdd",
		"FerryPolicy", uniqueKey(f.Name, f.Namespace),
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
		"FerryPolicy", uniqueKey(f.Name, f.Namespace),
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
		"FerryPolicy", uniqueKey(f.Name, f.Namespace),
	)

	c.mut.Lock()
	defer c.mut.Unlock()

	delete(c.cache, f.Name)

	cancel, ok := c.mapCancel[f.Name]
	if ok && cancel != nil {
		cancel()
	}

	delete(c.mapCancel, f.Name)
}

func getIPs(ctx context.Context, clientset *kubernetes.Clientset, route *v1alpha1.ClusterInformationSpecRoute) ([]string, error) {
	if route.IP != nil {
		return []string{*route.IP}, nil
	}
	if route.ServiceNamespace == nil {
		return nil, fmt.Errorf("ServiceNamespace is empty")
	}
	if route.ServiceName == nil {
		return nil, fmt.Errorf("ServiceName is empty")
	}
	ep, err := clientset.CoreV1().Endpoints(*route.ServiceNamespace).Get(ctx, *route.ServiceName, metav1.GetOptions{})
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
