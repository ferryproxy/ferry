package resource

import (
	"context"
	"fmt"
	"reflect"

	"github.com/ferry-proxy/api/apis/traffic/v1alpha2"
	versioned "github.com/ferry-proxy/client-go/generated/clientset/versioned"
	"github.com/ferry-proxy/ferry/pkg/consts"
	"github.com/ferry-proxy/ferry/pkg/ferry-controller/utils"
	"github.com/ferry-proxy/ferry/pkg/utils/objref"
	"github.com/go-logr/logr"
	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type ResourceBuilder interface {
	Build(proxy *Proxy, origin, destination objref.ObjectRef, spec *corev1.ServiceSpec) ([]Resourcer, error)
}

type ResourceBuilders []ResourceBuilder

func (r ResourceBuilders) Build(proxy *Proxy, origin, destination objref.ObjectRef, spec *corev1.ServiceSpec) ([]Resourcer, error) {
	var resourcers []Resourcer
	for _, i := range r {
		resourcer, err := i.Build(proxy, origin, destination, spec)
		if err != nil {
			return nil, err
		}
		resourcers = append(resourcers, resourcer...)
	}
	return resourcers, nil
}

type Proxy struct {
	RemotePrefix string
	Reverse      bool

	TunnelNamespace string

	ImportHubName string
	ExportHubName string

	Labels map[string]string

	ExportIngressAddress string
	ExportIdentity       string

	ImportIngressAddress string
	ImportIdentity       string

	ExportProxy []string
	ImportProxy []string

	GetPortFunc func(namespace, name string, port int32) int32
}

type Resourcer interface {
	utils.KMetadata
	Apply(ctx context.Context, clientset kubernetes.Interface) (err error)
	Delete(ctx context.Context, clientset kubernetes.Interface) (err error)
}

type Route struct {
	*v1alpha2.Route
}

func (rule *Route) Apply(ctx context.Context, clientset *versioned.Clientset) (err error) {
	logger := logr.FromContextOrDiscard(ctx)
	ori, err := clientset.
		TrafficV1alpha2().
		Routes(rule.Namespace).
		Get(ctx, rule.Name, metav1.GetOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			return fmt.Errorf("get mapping rule %s: %w", objref.KObj(rule), err)
		}
		logger.Info("Creating Service", "Service", objref.KObj(rule))
		_, err = clientset.
			TrafficV1alpha2().
			Routes(rule.Namespace).
			Create(ctx, rule.Route, metav1.CreateOptions{
				FieldManager: consts.LabelFerryManagedByValue,
			})
		if err != nil {
			return fmt.Errorf("create mapping rule %s: %w", objref.KObj(rule), err)
		}
	} else {
		_, err = clientset.
			TrafficV1alpha2().
			Routes(rule.Namespace).
			Update(ctx, ori, metav1.UpdateOptions{
				FieldManager: consts.LabelFerryManagedByValue,
			})
		if err != nil {
			return fmt.Errorf("update mapping rule %s: %w", objref.KObj(rule), err)
		}
	}
	return nil
}

func (rule *Route) Delete(ctx context.Context, clientset *versioned.Clientset) (err error) {
	logger := logr.FromContextOrDiscard(ctx)
	logger.Info("Deleting Service", "Service", objref.KObj(rule))

	err = clientset.
		TrafficV1alpha2().
		Routes(rule.Namespace).
		Delete(ctx, rule.Name, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("delete mapping rule  %s: %w", objref.KObj(rule), err)
	}
	return nil
}

type Service struct {
	*corev1.Service
}

func (s Service) Apply(ctx context.Context, clientset kubernetes.Interface) (err error) {
	logger := logr.FromContextOrDiscard(ctx)
	ori, err := clientset.
		CoreV1().
		Services(s.Namespace).
		Get(ctx, s.Name, metav1.GetOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			return fmt.Errorf("get service %s: %w", objref.KObj(s), err)
		}
		logger.Info("Creating Service", "Service", objref.KObj(s))
		_, err = clientset.
			CoreV1().
			Services(s.Namespace).
			Create(ctx, s.Service, metav1.CreateOptions{
				FieldManager: consts.LabelFerryManagedByValue,
			})
		if err != nil {
			return fmt.Errorf("create service %s: %w", objref.KObj(s), err)
		}
	} else {
		if ori.Labels[consts.LabelFerryManagedByKey] != consts.LabelFerryManagedByValue {
			return fmt.Errorf("service %s is not managed by ferry", objref.KObj(s))
		}
		if reflect.DeepEqual(ori.Spec.Ports, s.Spec.Ports) {
			return nil
		}

		copyLabel(ori.Labels, s.Labels)

		logger.Info("Update Service", "Service", objref.KObj(s))
		logger.Info(cmp.Diff(ori.Spec.Ports, s.Spec.Ports), "Service", objref.KObj(s))
		ori.Spec.Ports = s.Spec.Ports
		_, err = clientset.
			CoreV1().
			Services(s.Namespace).
			Update(ctx, ori, metav1.UpdateOptions{
				FieldManager: consts.LabelFerryManagedByValue,
			})
		if err != nil {
			return fmt.Errorf("update service %s: %w", objref.KObj(s), err)
		}
	}
	return nil
}

func (s Service) Delete(ctx context.Context, clientset kubernetes.Interface) (err error) {
	logger := logr.FromContextOrDiscard(ctx)
	logger.Info("Deleting Service", "Service", objref.KObj(s))

	err = clientset.
		CoreV1().
		Services(s.Namespace).
		Delete(ctx, s.Name, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("delete service %s: %w", objref.KObj(s), err)
	}
	return nil
}

type ConfigMap struct {
	*corev1.ConfigMap
}

func (s ConfigMap) Apply(ctx context.Context, clientset kubernetes.Interface) (err error) {
	logger := logr.FromContextOrDiscard(ctx)

	ori, err := clientset.
		CoreV1().
		ConfigMaps(s.Namespace).
		Get(ctx, s.Name, metav1.GetOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			return fmt.Errorf("get ConfigMap %s: %w", objref.KObj(s), err)
		}
		logger.Info("Creating ConfigMap", "ConfigMap", objref.KObj(s))
		_, err = clientset.
			CoreV1().
			ConfigMaps(s.Namespace).
			Create(ctx, s.ConfigMap, metav1.CreateOptions{
				FieldManager: consts.LabelFerryManagedByValue,
			})
		if err != nil {
			return fmt.Errorf("create ConfigMap %s: %w", objref.KObj(s), err)
		}
	} else {
		if ori.Labels[consts.LabelFerryManagedByKey] != consts.LabelFerryManagedByValue {
			return fmt.Errorf("configmap %s is not managed by ferry", objref.KObj(s))
		}

		if reflect.DeepEqual(ori.Data, s.Data) {
			return nil
		}

		copyLabel(ori.Labels, s.Labels)

		logger.Info("Update ConfigMap", "ConfigMap", objref.KObj(s))
		logger.Info(cmp.Diff(ori.Data, s.Data), "ConfigMap", objref.KObj(s))

		ori.Data = s.Data
		_, err = clientset.
			CoreV1().
			ConfigMaps(s.Namespace).
			Update(ctx, ori, metav1.UpdateOptions{
				FieldManager: consts.LabelFerryManagedByValue,
			})
		if err != nil {
			return fmt.Errorf("update ConfigMap %s: %w", objref.KObj(s), err)
		}
	}
	return nil
}

func (s ConfigMap) Delete(ctx context.Context, clientset kubernetes.Interface) (err error) {
	logger := logr.FromContextOrDiscard(ctx)
	logger.Info("Deleting ConfigMap", "ConfigMap", objref.KObj(s))

	err = clientset.
		CoreV1().
		ConfigMaps(s.Namespace).
		Delete(ctx, s.Name, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("delete ConfigMap %s: %w", objref.KObj(s), err)
	}

	return nil
}

func copyLabel(old, new map[string]string) {
	keys := []string{
		consts.LabelFerryExportedFromKey,
		consts.LabelFerryExportedFromNamespaceKey,
		consts.LabelFerryExportedFromNameKey,
		consts.LabelFerryExportedFromPortsKey,
		consts.LabelFerryImportedToKey,
		consts.LabelFerryTunnelKey,
	}
	for _, key := range keys {
		if v, ok := new[key]; ok {
			old[key] = v
		} else {
			if _, ok := old[key]; ok {
				delete(old, key)
			}
		}
	}
}
