package controller

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"strconv"
	"sync"

	"github.com/ferry-proxy/api/apis/ferry/v1alpha1"
	versioned "github.com/ferry-proxy/client-go/generated/clientset/versioned"
	externalversions "github.com/ferry-proxy/client-go/generated/informers/externalversions"
	"github.com/ferry-proxy/ferry/pkg/client"
	"github.com/ferry-proxy/utils/objref"
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
)

type clusterInformationControllerConfig struct {
	Logger    logr.Logger
	Config    *restclient.Config
	Namespace string
	SyncFunc  func()
}
type clusterInformationController struct {
	mut                     sync.RWMutex
	ctx                     context.Context
	logger                  logr.Logger
	config                  *restclient.Config
	cacheClusterInformation map[string]*v1alpha1.ClusterInformation
	cacheClientset          map[string]*kubernetes.Clientset
	cacheService            map[string]*clusterServiceCache
	cacheTunnelPorts        map[string]*tunnelPorts
	syncFunc                func()
	namespace               string
}

func newClusterInformationController(conf *clusterInformationControllerConfig) *clusterInformationController {
	return &clusterInformationController{
		config:                  conf.Config,
		namespace:               conf.Namespace,
		logger:                  conf.Logger,
		syncFunc:                conf.SyncFunc,
		cacheClusterInformation: map[string]*v1alpha1.ClusterInformation{},
		cacheClientset:          map[string]*kubernetes.Clientset{},
		cacheService:            map[string]*clusterServiceCache{},
		cacheTunnelPorts:        map[string]*tunnelPorts{},
	}
}

func (c *clusterInformationController) Run(ctx context.Context) error {
	c.logger.Info("ClusterInformation controller started")
	defer c.logger.Info("ClusterInformation controller stopped")

	clientset, err := versioned.NewForConfig(c.config)
	if err != nil {
		return err
	}
	c.ctx = ctx
	informerFactory := externalversions.NewSharedInformerFactoryWithOptions(clientset, 0,
		externalversions.WithNamespace(c.namespace))
	informer := informerFactory.
		Ferry().
		V1alpha1().
		ClusterInformations().
		Informer()
	informer.AddEventHandler(c)
	informer.Run(ctx.Done())
	return nil
}

func (c *clusterInformationController) Clientset(name string) *kubernetes.Clientset {
	c.mut.RLock()
	defer c.mut.RUnlock()
	return c.cacheClientset[name]
}

func (c *clusterInformationController) ServiceCache(name string) *clusterServiceCache {
	c.mut.RLock()
	defer c.mut.RUnlock()
	return c.cacheService[name]
}

func (c *clusterInformationController) TunnelPorts(name string) *tunnelPorts {
	c.mut.RLock()
	defer c.mut.RUnlock()
	return c.cacheTunnelPorts[name]
}

func (c *clusterInformationController) OnAdd(obj interface{}) {
	f := obj.(*v1alpha1.ClusterInformation)
	f = f.DeepCopy()
	c.logger.Info("OnAdd",
		"ClusterInformation", objref.KObj(f),
	)

	c.mut.Lock()
	defer c.mut.Unlock()

	clientset, err := client.NewClientsetFromKubeconfig(f.Spec.Kubeconfig)
	if err != nil {
		c.logger.Error(err, "NewClientsetFromKubeconfig")
	} else {
		c.cacheClientset[f.Name] = clientset
	}

	c.cacheClusterInformation[f.Name] = f
	c.cacheTunnelPorts[f.Name] = newTunnelPorts(&tunnelPortsConfig{
		Logger: c.logger.WithName(f.Name),
	})

	clusterService := newClusterServiceCache(clusterServiceCacheConfig{
		Clientset: clientset,
		Logger:    c.logger.WithName(f.Name),
	})
	c.cacheService[f.Name] = clusterService

	err = clusterService.Start(c.ctx)
	if err != nil {
		c.logger.Error(err, "failed start cluster service cache")
	}

	c.syncFunc()
}

func (c *clusterInformationController) OnUpdate(oldObj, newObj interface{}) {
	f := newObj.(*v1alpha1.ClusterInformation)
	f = f.DeepCopy()
	c.logger.Info("OnUpdate",
		"ClusterInformation", objref.KObj(f),
	)

	c.mut.Lock()
	defer c.mut.Unlock()

	if !bytes.Equal(c.cacheClusterInformation[f.Name].Spec.Kubeconfig, f.Spec.Kubeconfig) {
		clientset, err := client.NewClientsetFromKubeconfig(f.Spec.Kubeconfig)
		if err != nil {
			c.logger.Error(err, "NewClientsetFromKubeconfig")
		} else {
			c.cacheClientset[f.Name] = clientset
			err := c.cacheService[f.Name].ResetClientset(clientset)
			if err != nil {
				c.logger.Error(err, "Reset clientset")
			}
		}
	}

	c.cacheClusterInformation[f.Name] = f

	c.syncFunc()
}

func (c *clusterInformationController) OnDelete(obj interface{}) {
	f := obj.(*v1alpha1.ClusterInformation)
	c.logger.Info("OnDelete",
		"ClusterInformation", objref.KObj(f),
	)

	c.mut.Lock()
	defer c.mut.Unlock()

	delete(c.cacheClientset, f.Name)
	delete(c.cacheClusterInformation, f.Name)
	delete(c.cacheTunnelPorts, f.Name)

	if c.cacheService[f.Name] != nil {
		c.cacheService[f.Name].Close()
	}
	delete(c.cacheService, f.Name)

	c.syncFunc()
}

func (c *clusterInformationController) Get(name string) *v1alpha1.ClusterInformation {
	c.mut.RLock()
	defer c.mut.RUnlock()
	return c.cacheClusterInformation[name]
}

func (c *clusterInformationController) proxy(ctx context.Context, proxy v1alpha1.Proxy) (string, error) {
	if proxy.Proxy != "" {
		return proxy.Proxy, nil
	}

	ip, err := c.GetIPs(ctx, proxy.ClusterName)
	if err != nil {
		return "", fmt.Errorf("failed get ip: %w", err)
	}

	port, err := c.GetPort(ctx, proxy.ClusterName)
	if err != nil {
		return "", fmt.Errorf("failed get port: %w", err)
	}

	return "ssh://" + net.JoinHostPort(ip[0], strconv.FormatInt(int64(port), 10)), nil
}

func (c *clusterInformationController) proxies(ctx context.Context, proxies []v1alpha1.Proxy) ([]string, error) {
	out := make([]string, 0, len(proxies))
	for _, proxy := range proxies {
		p, err := c.proxy(ctx, proxy)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

func (c *clusterInformationController) GetPort(ctx context.Context, clusterName string) (int32, error) {
	ci := c.Get(clusterName)
	if ci == nil {
		return 0, fmt.Errorf("not found cluster %s", clusterName)
	}
	if ci.Spec.Ingress == nil {
		return 0, fmt.Errorf("not ingress int cluster %s", clusterName)
	}

	route := ci.Spec.Ingress

	if route == nil {
		return 31087, nil
	}
	if route.Port != 0 {
		return route.Port, nil
	}
	if route.ServiceNamespace == "" && route.ServiceName == "" {
		return 31087, nil
	}
	if route.ServiceNamespace == "" {
		return 0, fmt.Errorf("ServiceNamespace is empty")
	}
	if route.ServiceName == "" {
		return 0, fmt.Errorf("ServiceName is empty")
	}

	clientset := c.Clientset(clusterName)
	if clientset == nil {
		return 0, fmt.Errorf("not found clientset on cluster %s", clusterName)
	}
	ep, err := clientset.
		CoreV1().
		Endpoints(route.ServiceNamespace).
		Get(ctx, route.ServiceName, metav1.GetOptions{})
	if err != nil {
		return 0, err
	}
	if len(ep.Subsets) == 0 {
		return 0, fmt.Errorf("Endpoints's Subsets is empty")
	}

	if len(ep.Subsets[0].Ports) == 0 {
		return 0, fmt.Errorf("Endpoints's Subsets[0].Ports is empty")
	}

	for _, port := range ep.Subsets[0].Ports {
		if port.Port != 0 {
			return port.Port, nil
		}
	}
	return 31087, nil
}

func (c *clusterInformationController) GetIPs(ctx context.Context, clusterName string) ([]string, error) {
	ci := c.Get(clusterName)
	if ci == nil {
		return nil, fmt.Errorf("not found cluster %s", clusterName)
	}
	if ci.Spec.Ingress == nil {
		return nil, fmt.Errorf("not ingress int cluster %s", clusterName)
	}

	route := ci.Spec.Ingress

	if route == nil {
		return nil, nil
	}
	if route.IP != "" {
		return []string{route.IP}, nil
	}
	if route.ServiceNamespace == "" && route.ServiceName == "" {
		return nil, nil
	}
	if route.ServiceNamespace == "" {
		return nil, fmt.Errorf("ServiceNamespace is empty")
	}
	if route.ServiceName == "" {
		return nil, fmt.Errorf("ServiceName is empty")
	}

	clientset := c.Clientset(clusterName)
	if clientset == nil {
		return nil, fmt.Errorf("not found clientset on cluster %s", clusterName)
	}
	ep, err := clientset.
		CoreV1().
		Endpoints(route.ServiceNamespace).
		Get(ctx, route.ServiceName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	if len(ep.Subsets) == 0 {
		return nil, fmt.Errorf("Endpoints's Subsets is empty")
	}

	if len(ep.Subsets[0].Addresses) == 0 {
		return nil, fmt.Errorf("Endpoints's Subsets[0].Addresses is empty")
	}

	ips := []string{}
	for _, address := range ep.Subsets[0].Addresses {
		if address.IP == "" {
			continue
		}
		ips = append(ips, address.IP)
	}
	return ips, nil
}
