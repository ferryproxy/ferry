package client

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/util/flowcontrol"
)

func NewClientsetFromKubeconfig(kubeconfig []byte) (kubernetes.Interface, error) {
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
