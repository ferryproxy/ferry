package router

import (
	"context"
	"fmt"
	"reflect"

	"github.com/ferry-proxy/ferry/pkg/consts"
	"github.com/ferry-proxy/ferry/pkg/utils"
	"github.com/go-logr/logr"
	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type ResourceBuilder interface {
	Build(proxy *Proxy, origin, destination utils.ObjectRef, spec *corev1.ServiceSpec) ([]Resourcer, error)
}

type ResourceBuilders []ResourceBuilder

func (r ResourceBuilders) Build(proxy *Proxy, origin, destination utils.ObjectRef, spec *corev1.ServiceSpec) ([]Resourcer, error) {
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

	ImportClusterName string
	ExportClusterName string

	Labels map[string]string

	ExportIngressIPs  []string
	ExportIngressPort int32

	ImportIngressIPs  []string
	ImportIngressPort int32

	ExportProxy []string
	ImportProxy []string

	GetPortFunc func(namespace, name string, port int32) int32
}

type Resourcer interface {
	Apply(ctx context.Context, clientset *kubernetes.Clientset) (err error)
	Delete(ctx context.Context, clientset *kubernetes.Clientset) (err error)
}

type Service struct {
	*corev1.Service
}

func (s Service) Apply(ctx context.Context, clientset *kubernetes.Clientset) (err error) {
	logger := logr.FromContextOrDiscard(ctx)
	ori, err := clientset.
		CoreV1().
		Services(s.Namespace).
		Get(ctx, s.Name, metav1.GetOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			return fmt.Errorf("get service %s: %w", utils.KObj(s), err)
		}
		logger.Info("Creating Service", "Service", utils.KObj(s))
		_, err = clientset.
			CoreV1().
			Services(s.Namespace).
			Create(ctx, s.Service, metav1.CreateOptions{
				FieldManager: consts.LabelFerryManagedByValue,
			})
		if err != nil {
			return fmt.Errorf("create service %s: %w", utils.KObj(s), err)
		}
	} else {
		if ori.Labels[consts.LabelFerryManagedByKey] != consts.LabelFerryManagedByValue {
			return fmt.Errorf("service %s is not managed by ferry", utils.KObj(s))
		}
		if reflect.DeepEqual(ori.Spec.Ports, s.Spec.Ports) {
			return nil
		}

		copyLabel(ori.Labels, s.Labels)

		logger.Info("Update Service", "Service", utils.KObj(s))
		logger.Info(cmp.Diff(ori.Spec.Ports, s.Spec.Ports), "Service", utils.KObj(s))
		ori.Spec.Ports = s.Spec.Ports
		_, err = clientset.
			CoreV1().
			Services(s.Namespace).
			Update(ctx, ori, metav1.UpdateOptions{
				FieldManager: consts.LabelFerryManagedByValue,
			})
		if err != nil {
			return fmt.Errorf("update service %s: %w", utils.KObj(s), err)
		}
	}
	return nil
}

func (s Service) Delete(ctx context.Context, clientset *kubernetes.Clientset) (err error) {
	logger := logr.FromContextOrDiscard(ctx)
	logger.Info("Deleting Service", "Service", utils.KObj(s))

	err = clientset.
		CoreV1().
		Services(s.Namespace).
		Delete(ctx, s.Name, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("delete service %s: %w", utils.KObj(s), err)
	}
	return nil
}

type ConfigMap struct {
	*corev1.ConfigMap
}

func (s ConfigMap) Apply(ctx context.Context, clientset *kubernetes.Clientset) (err error) {
	logger := logr.FromContextOrDiscard(ctx)

	ori, err := clientset.
		CoreV1().
		ConfigMaps(s.Namespace).
		Get(ctx, s.Name, metav1.GetOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			return fmt.Errorf("get ConfigMap %s: %w", utils.KObj(s), err)
		}
		logger.Info("Creating ConfigMap", "ConfigMap", utils.KObj(s))
		_, err = clientset.
			CoreV1().
			ConfigMaps(s.Namespace).
			Create(ctx, s.ConfigMap, metav1.CreateOptions{
				FieldManager: consts.LabelFerryManagedByValue,
			})
		if err != nil {
			return fmt.Errorf("create ConfigMap %s: %w", utils.KObj(s), err)
		}
	} else {
		if ori.Labels[consts.LabelFerryManagedByKey] != consts.LabelFerryManagedByValue {
			return fmt.Errorf("configmap %s is not managed by ferry", utils.KObj(s))
		}

		if reflect.DeepEqual(ori.Data, s.Data) {
			return nil
		}

		copyLabel(ori.Labels, s.Labels)

		logger.Info("Update ConfigMap", "ConfigMap", utils.KObj(s))
		logger.Info(cmp.Diff(ori.Data, s.Data), "ConfigMap", utils.KObj(s))

		ori.Data = s.Data
		_, err = clientset.
			CoreV1().
			ConfigMaps(s.Namespace).
			Update(ctx, ori, metav1.UpdateOptions{
				FieldManager: consts.LabelFerryManagedByValue,
			})
		if err != nil {
			return fmt.Errorf("update ConfigMap %s: %w", utils.KObj(s), err)
		}
	}
	return nil
}

func (s ConfigMap) Delete(ctx context.Context, clientset *kubernetes.Clientset) (err error) {
	logger := logr.FromContextOrDiscard(ctx)
	logger.Info("Deleting ConfigMap", "ConfigMap", utils.KObj(s))

	err = clientset.
		CoreV1().
		ConfigMaps(s.Namespace).
		Delete(ctx, s.Name, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("delete ConfigMap %s: %w", utils.KObj(s), err)
	}

	return nil
}

func CalculatePatchResources(older, newer []Resourcer) (updated, deleted []Resourcer) {
	if len(older) == 0 {
		return newer, nil
	}
	type meta interface {
		GetName() string
		GetNamespace() string
	}
	exist := map[string]Resourcer{}

	nameFunc := func(m meta) string {
		return fmt.Sprintf("%s/%s/%s", reflect.TypeOf(m).Name(), m.GetNamespace(), m.GetName())
	}
	for _, r := range older {
		m, ok := r.(meta)
		if !ok {
			continue
		}
		name := nameFunc(m)
		exist[name] = r
	}

	for _, r := range newer {
		m, ok := r.(meta)
		if !ok {
			continue
		}
		name := nameFunc(m)
		delete(exist, name)
	}
	for _, r := range exist {
		deleted = append(deleted, r)
	}
	return newer, deleted
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
