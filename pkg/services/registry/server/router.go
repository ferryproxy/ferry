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

package server

import (
	"context"
	"net/http"

	versioned "github.com/ferryproxy/client-go/generated/clientset/versioned"
	"github.com/go-logr/logr"
	"k8s.io/client-go/kubernetes"
	rest "k8s.io/client-go/rest"
)

func Serve(mux *http.ServeMux, logger logr.Logger, config *rest.Config, address string, getBindPort func(ctx context.Context) (int32, error)) error {
	kubeClientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}
	ferryClientset, err := versioned.NewForConfig(config)
	if err != nil {
		return err
	}
	c := &Controller{
		FerryClientset: ferryClientset,
		KubeClientset:  kubeClientset,
		TunnelAddress:  address,
		Logger:         logger,
		GetBindPort:    getBindPort,
	}
	mux.Handle("/hubs/", c)
	return nil
}
