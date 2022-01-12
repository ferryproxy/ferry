package router

import (
	"context"
	"fmt"
	"reflect"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	ferry = "ferry-controller"
)

type ResourceBuilder interface {
	Build(proxy *Proxy, destinationServices []*corev1.Service) ([]Resourcer, error)
}

type ResourceBuilders []ResourceBuilder

func (r ResourceBuilders) Build(proxy *Proxy, destinationServices []*corev1.Service) ([]Resourcer, error) {
	var resourcers []Resourcer
	for _, i := range r {
		resourcer, err := i.Build(proxy, destinationServices)
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

	ImportPortOffset int32
	ExportPortOffset int32

	Labels map[string]string

	InClusterEgressIPs []string

	ExportIngressIPs  []string
	ExportIngressPort int32

	ImportIngressIPs  []string
	ImportIngressPort int32
}

type Resourcer interface {
	Apply(ctx context.Context, clientset *kubernetes.Clientset) (err error)
	Delete(ctx context.Context, clientset *kubernetes.Clientset) (err error)
}

type ConfigMap struct {
	*corev1.ConfigMap
}

func (i ConfigMap) Apply(ctx context.Context, clientset *kubernetes.Clientset) (err error) {

	logr.FromContextOrDiscard(ctx).Info("Creating ConfigMap", "ConfigMap", i.ConfigMap)

	_, err = clientset.CoreV1().
		ConfigMaps(i.ConfigMap.Namespace).
		Create(ctx, i.ConfigMap, metav1.CreateOptions{
			FieldManager: ferry,
		})
	if err != nil {
		if errors.IsAlreadyExists(err) {
			i.ConfigMap.ResourceVersion = "0"
			_, err = clientset.CoreV1().
				ConfigMaps(i.ConfigMap.Namespace).
				Update(ctx, i.ConfigMap, metav1.UpdateOptions{
					FieldManager: ferry,
				})
			if err != nil {
				return fmt.Errorf("update ConfigMap %s.%s: %w", i.ConfigMap.Name, i.ConfigMap.Namespace, err)
			}
		} else {
			return fmt.Errorf("create ConfigMap %s.%s: %w", i.ConfigMap.Name, i.ConfigMap.Namespace, err)
		}
	}
	return nil
}

func (i ConfigMap) Delete(ctx context.Context, clientset *kubernetes.Clientset) (err error) {

	logr.FromContextOrDiscard(ctx).Info("Deleting ConfigMap", "ConfigMap", i.ConfigMap)

	err = clientset.CoreV1().
		ConfigMaps(i.ConfigMap.Namespace).
		Delete(ctx, i.ConfigMap.Name, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("delete ConfigMap %s.%s: %w", i.ConfigMap.Name, i.ConfigMap.Namespace, err)
	}
	return nil
}

type Service struct {
	*corev1.Service
}

func (s Service) Apply(ctx context.Context, clientset *kubernetes.Clientset) (err error) {

	logr.FromContextOrDiscard(ctx).Info("Creating Service", "Service", s.Service)

	ori, err := clientset.CoreV1().
		Services(s.Service.Namespace).
		Get(ctx, s.Service.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			_, err = clientset.CoreV1().
				Services(s.Service.Namespace).
				Create(ctx, s.Service, metav1.CreateOptions{
					FieldManager: ferry,
				})
			if err != nil {
				return fmt.Errorf("create service %s.%s: %w", s.Service.Name, s.Service.Namespace, err)
			}
		} else {
			return fmt.Errorf("get service %s.%s: %w", s.Service.Name, s.Service.Namespace, err)
		}
	} else {
		ori.Spec.Ports = s.Service.Spec.Ports
		ori.Labels = s.Service.Labels
		ori.Annotations = s.Service.Annotations
		_, err = clientset.CoreV1().
			Services(s.Service.Namespace).
			Update(ctx, ori, metav1.UpdateOptions{
				FieldManager: ferry,
			})
		if err != nil {
			return fmt.Errorf("update service %s.%s: %w", s.Service.Name, s.Service.Namespace, err)
		}
	}
	return nil
}

func (s Service) Delete(ctx context.Context, clientset *kubernetes.Clientset) (err error) {
	logr.FromContextOrDiscard(ctx).Info("Deleting Service", "Service", s.Service)

	err = clientset.CoreV1().
		Services(s.Service.Namespace).
		Delete(ctx, s.Service.Name, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("delete service %s.%s: %w", s.Service.Name, s.Service.Namespace, err)
	}
	return nil
}

type Endpoints struct {
	*corev1.Endpoints
}

func (e Endpoints) Apply(ctx context.Context, clientset *kubernetes.Clientset) (err error) {

	logr.FromContextOrDiscard(ctx).Info("Creating Endpoints", "Endpoints", e.Endpoints)

	_, err = clientset.CoreV1().
		Endpoints(e.Endpoints.Namespace).
		Create(ctx, e.Endpoints, metav1.CreateOptions{
			FieldManager: ferry,
		})
	if err != nil {
		if errors.IsAlreadyExists(err) {
			e.Endpoints.ResourceVersion = "0"
			_, err = clientset.CoreV1().
				Endpoints(e.Endpoints.Namespace).
				Update(ctx, e.Endpoints, metav1.UpdateOptions{
					FieldManager: ferry,
				})
			if err != nil {
				return fmt.Errorf("update endpoints %s.%s: %w", e.Endpoints.Name, e.Endpoints.Namespace, err)
			}
		} else {
			return fmt.Errorf("create endpoints %s.%s: %w", e.Endpoints.Name, e.Endpoints.Namespace, err)
		}
	}
	return nil
}

func (e Endpoints) Delete(ctx context.Context, clientset *kubernetes.Clientset) (err error) {

	logr.FromContextOrDiscard(ctx).Info("Deleting Endpoints", "Endpoints", e.Endpoints)

	err = clientset.CoreV1().
		Services(e.Endpoints.Namespace).
		Delete(ctx, e.Endpoints.Name, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("delete service %s.%s: %w", e.Endpoints.Name, e.Endpoints.Namespace, err)
	}
	if e.Endpoints != nil {
		err = clientset.CoreV1().
			Endpoints(e.Endpoints.Namespace).
			Delete(ctx, e.Endpoints.Name, metav1.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			return fmt.Errorf("delete endpoints %s.%s: %w", e.Endpoints.Name, e.Endpoints.Namespace, err)
		}
	}
	return nil
}

func BuildIPToEndpointAddress(ips []string) []corev1.EndpointAddress {
	eps := []corev1.EndpointAddress{}
	for _, ip := range ips {
		eps = append(eps, corev1.EndpointAddress{
			IP: ip,
		})
	}
	return eps
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
