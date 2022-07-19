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
	"github.com/ferryproxy/ferry/pkg/utils/objref"
	"github.com/go-logr/logr"
	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type Resourcer interface {
	objref.KMetadata
	Apply(ctx context.Context, clientset kubernetes.Interface) (err error)
	Delete(ctx context.Context, clientset kubernetes.Interface) (err error)
}

type Route struct {
	*v1alpha2.Route
}

func (r Route) Apply(ctx context.Context, clientset *versioned.Clientset) (err error) {
	logger := logr.FromContextOrDiscard(ctx)
	ori, err := clientset.
		TrafficV1alpha2().
		Routes(r.Namespace).
		Get(ctx, r.Name, metav1.GetOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			return fmt.Errorf("get route %s: %w", objref.KObj(r), err)
		}
		logger.Info("Creating Route", "Route", objref.KObj(r))
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

func (r Route) Delete(ctx context.Context, clientset *versioned.Clientset) (err error) {
	logger := logr.FromContextOrDiscard(ctx)
	logger.Info("Deleting Route", "Route", objref.KObj(r))

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
