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

package client

import (
	"context"
	"fmt"
	"reflect"

	"github.com/ferryproxy/api/apis/traffic/v1alpha2"
	"github.com/ferryproxy/ferry/pkg/consts"
	"github.com/ferryproxy/ferry/pkg/utils/objref"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type hub struct {
	*v1alpha2.Hub
}

func (r hub) Apply(ctx context.Context, clientset Interface) (err error) {
	logger := logr.FromContextOrDiscard(ctx)
	ori, err := clientset.
		Ferry().
		TrafficV1alpha2().
		Hubs(r.Namespace).
		Get(ctx, r.Name, metav1.GetOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			return fmt.Errorf("get hub %s: %w", objref.KObj(r), err)
		}
		logger.Info("Creating hub",
			"hub", objref.KObj(r),
		)
		_, err = clientset.
			Ferry().
			TrafficV1alpha2().
			Hubs(r.Namespace).
			Create(ctx, r.Hub, metav1.CreateOptions{
				FieldManager: consts.LabelFerryManagedByValue,
			})
		if err != nil {
			return fmt.Errorf("create hub %s: %w", objref.KObj(r), err)
		}
	} else {
		if reflect.DeepEqual(ori.Spec, r.Spec) {
			return nil
		}

		_, err = clientset.
			Ferry().
			TrafficV1alpha2().
			Hubs(r.Namespace).
			Update(ctx, ori, metav1.UpdateOptions{
				FieldManager: consts.LabelFerryManagedByValue,
			})
		if err != nil {
			return fmt.Errorf("update hub %s: %w", objref.KObj(r), err)
		}
	}
	return nil
}

func (r hub) Delete(ctx context.Context, clientset Interface) (err error) {
	logger := logr.FromContextOrDiscard(ctx)
	logger.Info("Deleting hub",
		"hub", objref.KObj(r),
	)

	err = clientset.
		Ferry().
		TrafficV1alpha2().
		Hubs(r.Namespace).
		Delete(ctx, r.Name, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("delete hub %s: %w", objref.KObj(r), err)
	}
	return nil
}

type routePolicy struct {
	*v1alpha2.RoutePolicy
}

func (r routePolicy) Apply(ctx context.Context, clientset Interface) (err error) {
	logger := logr.FromContextOrDiscard(ctx)
	ori, err := clientset.
		Ferry().
		TrafficV1alpha2().
		RoutePolicies(r.Namespace).
		Get(ctx, r.Name, metav1.GetOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			return fmt.Errorf("get RoutePolicies %s: %w", objref.KObj(r), err)
		}
		logger.Info("Creating routePolicy",
			"routePolicy", objref.KObj(r),
		)
		_, err = clientset.
			Ferry().
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
			Ferry().
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

func (r routePolicy) Delete(ctx context.Context, clientset Interface) (err error) {
	logger := logr.FromContextOrDiscard(ctx)
	logger.Info("Deleting RoutePolicies",
		"routePolicy", objref.KObj(r),
	)

	err = clientset.
		Ferry().
		TrafficV1alpha2().
		RoutePolicies(r.Namespace).
		Delete(ctx, r.Name, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("delete RoutePolicies %s: %w", objref.KObj(r), err)
	}
	return nil
}

type route struct {
	*v1alpha2.Route
}

func (r route) Apply(ctx context.Context, clientset Interface) (err error) {
	logger := logr.FromContextOrDiscard(ctx)
	ori, err := clientset.
		Ferry().
		TrafficV1alpha2().
		Routes(r.Namespace).
		Get(ctx, r.Name, metav1.GetOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			return fmt.Errorf("get route %s: %w", objref.KObj(r), err)
		}
		logger.Info("Creating route",
			"route", objref.KObj(r),
		)
		_, err = clientset.
			Ferry().
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
			Ferry().
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

func (r route) Delete(ctx context.Context, clientset Interface) (err error) {
	logger := logr.FromContextOrDiscard(ctx)
	logger.Info("Deleting route",
		"route", objref.KObj(r),
	)

	err = clientset.
		Ferry().
		TrafficV1alpha2().
		Routes(r.Namespace).
		Delete(ctx, r.Name, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("delete route %s: %w", objref.KObj(r), err)
	}
	return nil
}

type service struct {
	*corev1.Service
}

func (s service) Apply(ctx context.Context, clientset Interface) (err error) {
	logger := logr.FromContextOrDiscard(ctx)
	ori, err := clientset.
		Kubernetes().
		CoreV1().
		Services(s.Namespace).
		Get(ctx, s.Name, metav1.GetOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			return fmt.Errorf("get service %s: %w", objref.KObj(s), err)
		}
		logger.Info("Creating service",
			"service", objref.KObj(s),
		)
		_, err = clientset.
			Kubernetes().
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
			Kubernetes().
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

func (s service) Delete(ctx context.Context, clientset Interface) (err error) {
	logger := logr.FromContextOrDiscard(ctx)
	logger.Info("Deleting service",
		"service", objref.KObj(s),
	)

	err = clientset.
		Kubernetes().
		CoreV1().
		Services(s.Namespace).
		Delete(ctx, s.Name, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("delete service %s: %w", objref.KObj(s), err)
	}
	return nil
}

type endpoints struct {
	*corev1.Endpoints
}

func (s endpoints) Apply(ctx context.Context, clientset Interface) (err error) {
	logger := logr.FromContextOrDiscard(ctx)
	ori, err := clientset.
		Kubernetes().
		CoreV1().
		Endpoints(s.Namespace).
		Get(ctx, s.Name, metav1.GetOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			return fmt.Errorf("get endpoints %s: %w", objref.KObj(s), err)
		}
		logger.Info("Creating endpoints",
			"endpoints", objref.KObj(s),
		)
		_, err = clientset.
			Kubernetes().
			CoreV1().
			Endpoints(s.Namespace).
			Create(ctx, s.Endpoints, metav1.CreateOptions{
				FieldManager: consts.LabelFerryManagedByValue,
			})
		if err != nil {
			return fmt.Errorf("create endpoints %s: %w", objref.KObj(s), err)
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
			Kubernetes().
			CoreV1().
			Endpoints(s.Namespace).
			Update(ctx, ori, metav1.UpdateOptions{
				FieldManager: consts.LabelFerryManagedByValue,
			})
		if err != nil {
			return fmt.Errorf("update endpoints %s: %w", objref.KObj(s), err)
		}
	}
	return nil
}

func (s endpoints) Delete(ctx context.Context, clientset Interface) (err error) {
	logger := logr.FromContextOrDiscard(ctx)
	logger.Info("Deleting endpoints",
		"endpoints", objref.KObj(s),
	)

	err = clientset.
		Kubernetes().
		CoreV1().
		Endpoints(s.Namespace).
		Delete(ctx, s.Name, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("delete endpoints %s: %w", objref.KObj(s), err)
	}
	return nil
}

type configMap struct {
	*corev1.ConfigMap
}

func (s configMap) Apply(ctx context.Context, clientset Interface) (err error) {
	logger := logr.FromContextOrDiscard(ctx)

	ori, err := clientset.
		Kubernetes().
		CoreV1().
		ConfigMaps(s.Namespace).
		Get(ctx, s.Name, metav1.GetOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			return fmt.Errorf("get configMap %s: %w", objref.KObj(s), err)
		}
		logger.Info("Creating configMap",
			"configMap", objref.KObj(s),
		)
		_, err = clientset.
			Kubernetes().
			CoreV1().
			ConfigMaps(s.Namespace).
			Create(ctx, s.ConfigMap, metav1.CreateOptions{
				FieldManager: consts.LabelFerryManagedByValue,
			})
		if err != nil {
			return fmt.Errorf("create configMap %s: %w", objref.KObj(s), err)
		}
	} else {
		if reflect.DeepEqual(ori.Data, s.Data) {
			return nil
		}

		copyLabel(ori.Labels, s.Labels)

		logger.Info("Update configMap",
			"configMap", objref.KObj(s),
		)
		ori.Data = s.Data
		_, err = clientset.
			Kubernetes().
			CoreV1().
			ConfigMaps(s.Namespace).
			Update(ctx, ori, metav1.UpdateOptions{
				FieldManager: consts.LabelFerryManagedByValue,
			})
		if err != nil {
			return fmt.Errorf("update configMap %s: %w", objref.KObj(s), err)
		}
	}
	return nil
}

func (s configMap) Delete(ctx context.Context, clientset Interface) (err error) {
	logger := logr.FromContextOrDiscard(ctx)
	logger.Info("Deleting configMap",
		"configMap", objref.KObj(s),
	)

	err = clientset.
		Kubernetes().
		CoreV1().
		ConfigMaps(s.Namespace).
		Delete(ctx, s.Name, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("delete configMap %s: %w", objref.KObj(s), err)
	}

	return nil
}

type secret struct {
	*corev1.Secret
}

func (s secret) Apply(ctx context.Context, clientset Interface) (err error) {
	logger := logr.FromContextOrDiscard(ctx)

	ori, err := clientset.
		Kubernetes().
		CoreV1().
		Secrets(s.Namespace).
		Get(ctx, s.Name, metav1.GetOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			return fmt.Errorf("get secret %s: %w", objref.KObj(s), err)
		}
		logger.Info("Creating secret",
			"secret", objref.KObj(s),
		)
		_, err = clientset.
			Kubernetes().
			CoreV1().
			Secrets(s.Namespace).
			Create(ctx, s.Secret, metav1.CreateOptions{
				FieldManager: consts.LabelFerryManagedByValue,
			})
		if err != nil {
			return fmt.Errorf("create secret %s: %w", objref.KObj(s), err)
		}
	} else {
		if reflect.DeepEqual(ori.Data, s.Data) {
			return nil
		}

		copyLabel(ori.Labels, s.Labels)

		logger.Info("Update secret",
			"secret", objref.KObj(s),
		)

		ori.Data = s.Data
		_, err = clientset.
			Kubernetes().
			CoreV1().
			Secrets(s.Namespace).
			Update(ctx, ori, metav1.UpdateOptions{
				FieldManager: consts.LabelFerryManagedByValue,
			})
		if err != nil {
			return fmt.Errorf("update secret %s: %w", objref.KObj(s), err)
		}
	}
	return nil
}

func (s secret) Delete(ctx context.Context, clientset Interface) (err error) {
	logger := logr.FromContextOrDiscard(ctx)
	logger.Info("Deleting secret",
		"secret", objref.KObj(s),
	)

	err = clientset.
		Kubernetes().
		CoreV1().
		Secrets(s.Namespace).
		Delete(ctx, s.Name, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("delete secret %s: %w", objref.KObj(s), err)
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
