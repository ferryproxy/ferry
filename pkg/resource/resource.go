/*
Copyright 2022 FerryProxy Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package resource

import (
	"context"
	"fmt"
	"reflect"

	"github.com/ferryproxy/api/apis/traffic/v1alpha2"
	versioned "github.com/ferryproxy/client-go/generated/clientset/versioned"
	"github.com/ferryproxy/ferry/pkg/consts"
	"github.com/ferryproxy/ferry/pkg/utils/encoding"
	"github.com/ferryproxy/ferry/pkg/utils/objref"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
)

type Resourcer interface {
	objref.KMetadata
	Original() objref.KMetadata
	Apply(ctx context.Context, clientset kubernetes.Interface) (err error)
	Delete(ctx context.Context, clientset kubernetes.Interface) (err error)
}

type Hub struct {
	*v1alpha2.Hub
}

func (r Hub) Original() objref.KMetadata {
	return r.Hub
}

func (r Hub) Apply(ctx context.Context, clientset versioned.Interface) (err error) {
	logger := logr.FromContextOrDiscard(ctx)
	ori, err := clientset.
		TrafficV1alpha2().
		Hubs(r.Namespace).
		Get(ctx, r.Name, metav1.GetOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			return fmt.Errorf("get Hub %s: %w", objref.KObj(r), err)
		}
		logger.Info("Creating Hub",
			"hub", objref.KObj(r),
		)
		_, err = clientset.
			TrafficV1alpha2().
			Hubs(r.Namespace).
			Create(ctx, r.Hub, metav1.CreateOptions{
				FieldManager: consts.LabelFerryManagedByValue,
			})
		if err != nil {
			return fmt.Errorf("create Hub %s: %w", objref.KObj(r), err)
		}
	} else {
		if reflect.DeepEqual(ori.Spec, r.Spec) {
			return nil
		}

		_, err = clientset.
			TrafficV1alpha2().
			Hubs(r.Namespace).
			Update(ctx, ori, metav1.UpdateOptions{
				FieldManager: consts.LabelFerryManagedByValue,
			})
		if err != nil {
			return fmt.Errorf("update Hub %s: %w", objref.KObj(r), err)
		}
	}
	return nil
}

func (r Hub) Delete(ctx context.Context, clientset versioned.Interface) (err error) {
	logger := logr.FromContextOrDiscard(ctx)
	logger.Info("Deleting Hub",
		"hub", objref.KObj(r),
	)

	err = clientset.
		TrafficV1alpha2().
		Hubs(r.Namespace).
		Delete(ctx, r.Name, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("delete Hub %s: %w", objref.KObj(r), err)
	}
	return nil
}

type RoutePolicy struct {
	*v1alpha2.RoutePolicy
}

func (r RoutePolicy) Original() objref.KMetadata {
	return r.RoutePolicy
}

func (r RoutePolicy) Apply(ctx context.Context, clientset versioned.Interface) (err error) {
	logger := logr.FromContextOrDiscard(ctx)
	ori, err := clientset.
		TrafficV1alpha2().
		RoutePolicies(r.Namespace).
		Get(ctx, r.Name, metav1.GetOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			return fmt.Errorf("get RoutePolicies %s: %w", objref.KObj(r), err)
		}
		logger.Info("Creating RoutePolicy",
			"routePolicy", objref.KObj(r),
		)
		_, err = clientset.
			TrafficV1alpha2().
			RoutePolicies(r.Namespace).
			Create(ctx, r.RoutePolicy, metav1.CreateOptions{
				FieldManager: consts.LabelFerryManagedByValue,
			})
		if err != nil {
			return fmt.Errorf("create RoutePolicies %s: %w", objref.KObj(r), err)
		}
	} else {
		if reflect.DeepEqual(ori.Spec, r.Spec) {
			return nil
		}

		_, err = clientset.
			TrafficV1alpha2().
			RoutePolicies(r.Namespace).
			Update(ctx, ori, metav1.UpdateOptions{
				FieldManager: consts.LabelFerryManagedByValue,
			})
		if err != nil {
			return fmt.Errorf("update RoutePolicies %s: %w", objref.KObj(r), err)
		}
	}
	return nil
}

func (r RoutePolicy) Delete(ctx context.Context, clientset versioned.Interface) (err error) {
	logger := logr.FromContextOrDiscard(ctx)
	logger.Info("Deleting RoutePolicies",
		"routePolicy", objref.KObj(r),
	)

	err = clientset.
		TrafficV1alpha2().
		RoutePolicies(r.Namespace).
		Delete(ctx, r.Name, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("delete RoutePolicies %s: %w", objref.KObj(r), err)
	}
	return nil
}

type Route struct {
	*v1alpha2.Route
}

func (r Route) Original() objref.KMetadata {
	return r.Route
}

func (r Route) Apply(ctx context.Context, clientset versioned.Interface) (err error) {
	logger := logr.FromContextOrDiscard(ctx)
	ori, err := clientset.
		TrafficV1alpha2().
		Routes(r.Namespace).
		Get(ctx, r.Name, metav1.GetOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			return fmt.Errorf("get route %s: %w", objref.KObj(r), err)
		}
		logger.Info("Creating Route",
			"route", objref.KObj(r),
		)
		_, err = clientset.
			TrafficV1alpha2().
			Routes(r.Namespace).
			Create(ctx, r.Route, metav1.CreateOptions{
				FieldManager: consts.LabelFerryManagedByValue,
			})
		if err != nil {
			return fmt.Errorf("create route %s: %w", objref.KObj(r), err)
		}
	} else {
		if reflect.DeepEqual(ori.Spec, r.Spec) {
			return nil
		}

		_, err = clientset.
			TrafficV1alpha2().
			Routes(r.Namespace).
			Update(ctx, ori, metav1.UpdateOptions{
				FieldManager: consts.LabelFerryManagedByValue,
			})
		if err != nil {
			return fmt.Errorf("update route %s: %w", objref.KObj(r), err)
		}
	}
	return nil
}

func (r Route) Delete(ctx context.Context, clientset versioned.Interface) (err error) {
	logger := logr.FromContextOrDiscard(ctx)
	logger.Info("Deleting Route",
		"route", objref.KObj(r),
	)

	err = clientset.
		TrafficV1alpha2().
		Routes(r.Namespace).
		Delete(ctx, r.Name, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("delete route %s: %w", objref.KObj(r), err)
	}
	return nil
}

type Service struct {
	*corev1.Service
}

func (s Service) Original() objref.KMetadata {
	return s.Service
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
		logger.Info("Creating Service",
			"service", objref.KObj(s),
		)
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
		if ori.Labels[consts.LabelGeneratedKey] == "" {
			return fmt.Errorf("service %s is not managed by ferry", objref.KObj(s))
		}
		if reflect.DeepEqual(ori.Spec.Ports, s.Spec.Ports) {
			return nil
		}

		copyLabel(ori.Labels, s.Labels)

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
	logger.Info("Deleting Service",
		"service", objref.KObj(s),
	)

	err = clientset.
		CoreV1().
		Services(s.Namespace).
		Delete(ctx, s.Name, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("delete service %s: %w", objref.KObj(s), err)
	}
	return nil
}

type Endpoints struct {
	*corev1.Endpoints
}

func (s Endpoints) Original() objref.KMetadata {
	return s.Endpoints
}

func (s Endpoints) Apply(ctx context.Context, clientset kubernetes.Interface) (err error) {
	logger := logr.FromContextOrDiscard(ctx)
	ori, err := clientset.
		CoreV1().
		Endpoints(s.Namespace).
		Get(ctx, s.Name, metav1.GetOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			return fmt.Errorf("get Endpoints %s: %w", objref.KObj(s), err)
		}
		logger.Info("Creating Endpoints",
			"endpoints", objref.KObj(s),
		)
		_, err = clientset.
			CoreV1().
			Endpoints(s.Namespace).
			Create(ctx, s.Endpoints, metav1.CreateOptions{
				FieldManager: consts.LabelFerryManagedByValue,
			})
		if err != nil {
			return fmt.Errorf("create Endpoints %s: %w", objref.KObj(s), err)
		}
	} else {
		if ori.Labels[consts.LabelGeneratedKey] == "" {
			return fmt.Errorf("endpoints %s is not managed by ferry", objref.KObj(s))
		}
		if reflect.DeepEqual(ori.Subsets, s.Subsets) {
			return nil
		}
		ori.Subsets = s.Subsets
		_, err = clientset.
			CoreV1().
			Endpoints(s.Namespace).
			Update(ctx, ori, metav1.UpdateOptions{
				FieldManager: consts.LabelFerryManagedByValue,
			})
		if err != nil {
			return fmt.Errorf("update Endpoints %s: %w", objref.KObj(s), err)
		}
	}
	return nil
}

func (s Endpoints) Delete(ctx context.Context, clientset kubernetes.Interface) (err error) {
	logger := logr.FromContextOrDiscard(ctx)
	logger.Info("Deleting Endpoints",
		"endpoints", objref.KObj(s),
	)

	err = clientset.
		CoreV1().
		Endpoints(s.Namespace).
		Delete(ctx, s.Name, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("delete Endpoints %s: %w", objref.KObj(s), err)
	}
	return nil
}

type ConfigMap struct {
	*corev1.ConfigMap
}

func (s ConfigMap) Original() objref.KMetadata {
	return s.ConfigMap
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
		logger.Info("Creating ConfigMap",
			"configMap", objref.KObj(s),
		)
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
		if reflect.DeepEqual(ori.Data, s.Data) {
			return nil
		}

		copyLabel(ori.Labels, s.Labels)

		logger.Info("Update ConfigMap",
			"configMap", objref.KObj(s),
		)
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
	logger.Info("Deleting ConfigMap",
		"configMap", objref.KObj(s),
	)

	err = clientset.
		CoreV1().
		ConfigMaps(s.Namespace).
		Delete(ctx, s.Name, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("delete ConfigMap %s: %w", objref.KObj(s), err)
	}

	return nil
}

type Secret struct {
	*corev1.Secret
}

func (s Secret) Original() objref.KMetadata {
	return s.Secret
}

func (s Secret) Apply(ctx context.Context, clientset kubernetes.Interface) (err error) {
	logger := logr.FromContextOrDiscard(ctx)

	ori, err := clientset.
		CoreV1().
		Secrets(s.Namespace).
		Get(ctx, s.Name, metav1.GetOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			return fmt.Errorf("get Secret %s: %w", objref.KObj(s), err)
		}
		logger.Info("Creating Secret",
			"secret", objref.KObj(s),
		)
		_, err = clientset.
			CoreV1().
			Secrets(s.Namespace).
			Create(ctx, s.Secret, metav1.CreateOptions{
				FieldManager: consts.LabelFerryManagedByValue,
			})
		if err != nil {
			return fmt.Errorf("create Secret %s: %w", objref.KObj(s), err)
		}
	} else {
		if reflect.DeepEqual(ori.Data, s.Data) {
			return nil
		}

		copyLabel(ori.Labels, s.Labels)

		logger.Info("Update Secret",
			"secret", objref.KObj(s),
		)

		ori.Data = s.Data
		_, err = clientset.
			CoreV1().
			Secrets(s.Namespace).
			Update(ctx, ori, metav1.UpdateOptions{
				FieldManager: consts.LabelFerryManagedByValue,
			})
		if err != nil {
			return fmt.Errorf("update Secret %s: %w", objref.KObj(s), err)
		}
	}
	return nil
}

func (s Secret) Delete(ctx context.Context, clientset kubernetes.Interface) (err error) {
	logger := logr.FromContextOrDiscard(ctx)
	logger.Info("Deleting Secret",
		"secret", objref.KObj(s),
	)

	err = clientset.
		CoreV1().
		Secrets(s.Namespace).
		Delete(ctx, s.Name, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("delete Secret %s: %w", objref.KObj(s), err)
	}

	return nil
}

func copyLabel(old, new map[string]string) {
	keys := []string{
		consts.LabelFerryExportedFromKey,
		consts.LabelFerryImportedToKey,
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

func MarshalYAML(resources ...Resourcer) ([]byte, error) {
	objs := make([]runtime.Object, 0, len(resources))
	for _, resource := range resources {
		obj, ok := resource.Original().(runtime.Object)
		if !ok {
			return nil, fmt.Errorf("failed convert to runtime.Object")
		}
		objs = append(objs, obj)
	}

	return encoding.MarshalYAML(objs...)
}

func MarshalJSON(resources ...Resourcer) ([]byte, error) {
	objs := make([]runtime.Object, 0, len(resources))
	for _, resource := range resources {
		obj, ok := resource.Original().(runtime.Object)
		if !ok {
			return nil, fmt.Errorf("failed convert to runtime.Object")
		}
		objs = append(objs, obj)
	}

	return encoding.MarshalJSON(objs...)
}
