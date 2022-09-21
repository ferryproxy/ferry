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

	"github.com/ferryproxy/api/apis/traffic/v1alpha2"
	versioned "github.com/ferryproxy/client-go/generated/clientset/versioned"
	"github.com/ferryproxy/ferry/pkg/utils/objref"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type clientset struct {
	kubeClientset  kubernetes.Interface
	ferryClientset versioned.Interface
}

type Interface interface {
	Kubernetes() kubernetes.Interface
	Ferry() versioned.Interface
}

func NewForConfig(conf *rest.Config) (Interface, error) {
	kubeClientset, err := kubernetes.NewForConfig(conf)
	if err != nil {
		return nil, err
	}
	ferryClientset, err := versioned.NewForConfig(conf)
	if err != nil {
		return nil, err
	}
	return &clientset{
		kubeClientset:  kubeClientset,
		ferryClientset: ferryClientset,
	}, nil
}

func (c *clientset) Kubernetes() kubernetes.Interface {
	return c.kubeClientset
}

func (c *clientset) Ferry() versioned.Interface {
	return c.ferryClientset
}

func Apply(ctx context.Context, c Interface, obj objref.KMetadata) error {
	switch o := obj.(type) {
	case *corev1.ConfigMap:
		return configMap{o}.Apply(ctx, c)
	case *corev1.Secret:
		return secret{o}.Apply(ctx, c)
	case *corev1.Service:
		return service{o}.Apply(ctx, c)
	case *corev1.Endpoints:
		return endpoints{o}.Apply(ctx, c)
	case *v1alpha2.Hub:
		return hub{o}.Apply(ctx, c)
	case *v1alpha2.RoutePolicy:
		return routePolicy{o}.Apply(ctx, c)
	case *v1alpha2.Route:
		return route{o}.Apply(ctx, c)
	default:
		return fmt.Errorf("unsupport type")
	}
}

func Delete(ctx context.Context, c Interface, obj objref.KMetadata) error {
	switch o := obj.(type) {
	case *corev1.ConfigMap:
		return configMap{o}.Delete(ctx, c)
	case *corev1.Secret:
		return secret{o}.Delete(ctx, c)
	case *corev1.Service:
		return service{o}.Delete(ctx, c)
	case *corev1.Endpoints:
		return endpoints{o}.Delete(ctx, c)
	case *v1alpha2.Hub:
		return hub{o}.Delete(ctx, c)
	case *v1alpha2.RoutePolicy:
		return routePolicy{o}.Delete(ctx, c)
	case *v1alpha2.Route:
		return route{o}.Delete(ctx, c)
	default:
		return fmt.Errorf("unsupport type")
	}
}
