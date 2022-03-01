package controller

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	"github.com/ferry-proxy/api/apis/ferry/v1alpha1"
	versioned "github.com/ferry-proxy/client-go/generated/clientset/versioned"
	externalversions "github.com/ferry-proxy/client-go/generated/informers/externalversions"
	"github.com/ferry-proxy/utils/objref"
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	restclient "k8s.io/client-go/rest"
)

type ferryPolicyControllerConfig struct {
	Logger    logr.Logger
	Config    *restclient.Config
	Namespace string
	SyncFunc  func()
}

type ferryPolicyController struct {
	ctx       context.Context
	mut       sync.RWMutex
	config    *restclient.Config
	clientset *versioned.Clientset
	cache     map[string]*v1alpha1.FerryPolicy
	namespace string
	syncFunc  func()
	logger    logr.Logger
}

func newFerryPolicyController(conf *ferryPolicyControllerConfig) *ferryPolicyController {
	return &ferryPolicyController{
		config:    conf.Config,
		namespace: conf.Namespace,
		logger:    conf.Logger,
		syncFunc:  conf.SyncFunc,
		cache:     map[string]*v1alpha1.FerryPolicy{},
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
	c.clientset = clientset
	c.ctx = ctx
	informerFactory := externalversions.NewSharedInformerFactoryWithOptions(clientset, 0,
		externalversions.WithNamespace(c.namespace))
	informer := informerFactory.
		Ferry().
		V1alpha1().
		FerryPolicies().
		Informer()
	informer.AddEventHandler(c)
	informer.Run(ctx.Done())
	return nil
}

func (c *ferryPolicyController) UpdateStatus(name string) error {
	c.mut.RLock()
	defer c.mut.RUnlock()

	fp := c.cache[name]
	if fp == nil {
		return fmt.Errorf("not found FerryPolicy %s", name)
	}

	fp = fp.DeepCopy()

	fp.Status.LastSynchronizationTimestamp = metav1.Now()

	_, err := c.clientset.
		FerryV1alpha1().
		FerryPolicies(c.namespace).
		UpdateStatus(c.ctx, fp, metav1.UpdateOptions{})
	return err
}

func (c *ferryPolicyController) OnAdd(obj interface{}) {
	f := obj.(*v1alpha1.FerryPolicy)
	f = f.DeepCopy()
	c.logger.Info("OnAdd",
		"FerryPolicy", objref.KObj(f),
	)

	c.mut.Lock()
	defer c.mut.Unlock()

	c.cache[f.Name] = f

	c.syncFunc()
}

func (c *ferryPolicyController) OnUpdate(oldObj, newObj interface{}) {
	f := newObj.(*v1alpha1.FerryPolicy)
	f = f.DeepCopy()
	c.logger.Info("OnUpdate",
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
}

func (c *ferryPolicyController) OnDelete(obj interface{}) {
	f := obj.(*v1alpha1.FerryPolicy)
	c.logger.Info("OnDelete",
		"FerryPolicy", objref.KObj(f),
	)

	c.mut.Lock()
	defer c.mut.Unlock()

	delete(c.cache, f.Name)

	c.syncFunc()
}
