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
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/util/flowcontrol"
)

func NewRestConfigFromKubeconfig(kubeconfig []byte) (*rest.Config, error) {
	cfg, err := clientcmd.BuildConfigFromKubeconfigGetter("", func() (conf *clientcmdapi.Config, err error) {
		return clientcmd.Load(kubeconfig)
	})
	if err != nil {
		return nil, err
	}
	err = setConfigDefaults(cfg)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func NewClientsetFromKubeconfig(kubeconfig []byte) (kubernetes.Interface, error) {
	cfg, err := NewRestConfigFromKubeconfig(kubeconfig)
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(cfg)
}

func setConfigDefaults(config *rest.Config) error {
	config.GroupVersion = &schema.GroupVersion{Group: "", Version: "v1"}
	if config.APIPath == "" {
		config.APIPath = "/api"
	}
	//if config.NegotiatedSerializer == nil {
	//	config.NegotiatedSerializer = scheme.Codecs.WithoutConversion()
	//}
	config.RateLimiter = flowcontrol.NewFakeAlwaysRateLimiter()
	return rest.SetKubernetesDefaults(config)
}
