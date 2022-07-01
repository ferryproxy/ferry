package controller

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	"github.com/ferry-proxy/api/apis/ferry/v1alpha1"
	versioned "github.com/ferry-proxy/client-go/generated/clientset/versioned"
	externalversions "github.com/ferry-proxy/client-go/generated/informers/externalversions"
	"github.com/ferry-proxy/ferry/pkg/consts"
	"github.com/ferry-proxy/utils/objref"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	restclient "k8s.io/client-go/rest"
)

type mappingRuleControllerConfig struct {
	Logger    logr.Logger
	Config    *restclient.Config
	Namespace string
	SyncFunc  func()
}

type mappingRuleController struct {
	ctx       context.Context
	mut       sync.RWMutex
	config    *restclient.Config
	clientset *versioned.Clientset
	cache     map[string]*v1alpha1.MappingRule
	namespace string
	syncFunc  func()
	logger    logr.Logger
}

func newMappingRuleController(conf *mappingRuleControllerConfig) *mappingRuleController {
	return &mappingRuleController{
		config:    conf.Config,
		namespace: conf.Namespace,
		logger:    conf.Logger,
		syncFunc:  conf.SyncFunc,
		cache:     map[string]*v1alpha1.MappingRule{},
	}
}

func (c *mappingRuleController) List() []*v1alpha1.MappingRule {
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

func (c *mappingRuleController) Get(name string) *v1alpha1.MappingRule {
	c.mut.RLock()
	defer c.mut.RUnlock()
	return c.cache[name]
}

func (c *mappingRuleController) Run(ctx context.Context) error {
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
	informer.AddEventHandler(c)
	informer.Run(ctx.Done())
	return nil
}

func (c *mappingRuleController) UpdateStatus(name string, phase string) error {
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

func (c *mappingRuleController) OnAdd(obj interface{}) {
	f := obj.(*v1alpha1.MappingRule)
	f = f.DeepCopy()
	c.logger.Info("OnAdd",
		"MappingRule", objref.KObj(f),
	)

	c.mut.Lock()
	defer c.mut.Unlock()

	c.cache[f.Name] = f

	c.syncFunc()
}

func (c *mappingRuleController) OnUpdate(oldObj, newObj interface{}) {
	f := newObj.(*v1alpha1.MappingRule)
	f = f.DeepCopy()
	c.logger.Info("OnUpdate",
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

func (c *mappingRuleController) OnDelete(obj interface{}) {
	f := obj.(*v1alpha1.MappingRule)
	c.logger.Info("OnDelete",
		"MappingRule", objref.KObj(f),
	)

	c.mut.Lock()
	defer c.mut.Unlock()

	delete(c.cache, f.Name)

	c.syncFunc()
}

func (c *mappingRuleController) Apply(ctx context.Context, rule *v1alpha1.MappingRule) (err error) {
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

func (c *mappingRuleController) Delete(ctx context.Context, rule *v1alpha1.MappingRule) (err error) {
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
