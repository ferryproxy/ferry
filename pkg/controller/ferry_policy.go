package controller

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/ferry-proxy/api/apis/ferry/v1alpha1"
	versioned "github.com/ferry-proxy/client-go/generated/clientset/versioned"
	externalversions "github.com/ferry-proxy/client-go/generated/informers/externalversions"
	"github.com/ferry-proxy/ferry/pkg/router"
	"github.com/ferry-proxy/ferry/pkg/utils"
	"github.com/ferry-proxy/utils/objref"
	"github.com/ferry-proxy/utils/trybuffer"
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	restclient "k8s.io/client-go/rest"
)

type ferryPolicyControllerConfig struct {
	Logger                       logr.Logger
	Config                       *restclient.Config
	ClusterInformationController *clusterInformationController
	Namespace                    string
}

type ferryPolicyController struct {
	ctx                          context.Context
	mut                          sync.RWMutex
	config                       *restclient.Config
	clientset                    *versioned.Clientset
	clusterInformationController *clusterInformationController
	cache                        map[string]*v1alpha1.FerryPolicy
	namespace                    string
	logger                       logr.Logger
	cacheFerryPolicyMappingRules []*v1alpha1.MappingRule
	try                          *trybuffer.TryBuffer
	syncFunc                     func()
}

func newFerryPolicyController(conf *ferryPolicyControllerConfig) *ferryPolicyController {
	return &ferryPolicyController{
		config:                       conf.Config,
		namespace:                    conf.Namespace,
		logger:                       conf.Logger,
		clusterInformationController: conf.ClusterInformationController,
		cache:                        map[string]*v1alpha1.FerryPolicy{},
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

	c.try = trybuffer.NewTryBuffer(func() {
		c.sync(ctx)
	}, time.Second)
	c.syncFunc = c.try.Try
	defer c.try.Close()
	informer.Run(ctx.Done())
	return nil
}

func (c *ferryPolicyController) UpdateStatus(name string, phase string) error {
	c.mut.RLock()
	defer c.mut.RUnlock()

	fp := c.cache[name]
	if fp == nil {
		return fmt.Errorf("not found FerryPolicy %s", name)
	}

	fp = fp.DeepCopy()

	fp.Status.LastSynchronizationTimestamp = metav1.Now()
	fp.Status.Phase = phase

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

func (c *ferryPolicyController) sync(ctx context.Context) {
	ferryPolicies := c.List()
	for _, policy := range ferryPolicies {
		err := c.UpdateStatus(policy.Name, "Working")
		if err != nil {
			c.logger.Error(err, "failed to update status")
		}
	}
	defer func() {
		for _, policy := range ferryPolicies {
			err := c.UpdateStatus(policy.Name, "Worked")
			if err != nil {
				c.logger.Error(err, "failed to update status")
			}
		}
	}()

	mappingRules := c.clusterInformationController.PoliciesToRules(ferryPolicies)

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
}
