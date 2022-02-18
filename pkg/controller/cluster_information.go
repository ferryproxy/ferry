package controller

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"reflect"
	"strconv"
	"sync"
	"time"

	"github.com/ferry-proxy/api/apis/ferry/v1alpha1"
	versioned "github.com/ferry-proxy/client-go/generated/clientset/versioned"
	externalversions "github.com/ferry-proxy/client-go/generated/informers/externalversions"
	"github.com/ferry-proxy/ferry/pkg/client"
	"github.com/ferry-proxy/ferry/pkg/utils"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
)

type clusterInformationControllerConfig struct {
	Logger    logr.Logger
	Config    *restclient.Config
	Namespace string
	SyncFunc  func(context.Context, string)
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
	cacheEgressWatchCancel  map[string]func()
	syncFunc                func(context.Context, string)
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
		cacheEgressWatchCancel:  map[string]func(){},
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
	informer := informerFactory.Ferry().
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

func (c *clusterInformationController) setupWatchEgress(ctx context.Context, ci *v1alpha1.ClusterInformation) {
	if c.syncFunc == nil {
		return
	}
	clientset := c.cacheClientset[ci.Name]
	if clientset == nil {
		return
	}

	if cluster := c.cacheClusterInformation[ci.Name]; cluster != nil &&
		needWatchEgress(cluster.Spec.Egress) &&
		reflect.DeepEqual(cluster.Spec.Egress, ci.Spec.Egress) {
		return
	}

	egress := ci.Spec.Egress

	if !needWatchEgress(egress) {
		if last, ok := c.cacheEgressWatchCancel[ci.Name]; last != nil && ok {
			last()
			delete(c.cacheEgressWatchCancel, ci.Name)
		}
		return
	}
	ctx, cancel := context.WithCancel(ctx)
	log := c.logger.WithName("watch-egress")
	fieldSelector := fmt.Sprintf("metadata.name=%s", egress.ServiceName)
	watch, err := clientset.CoreV1().Endpoints(egress.ServiceNamespace).Watch(ctx, metav1.ListOptions{
		FieldSelector: fieldSelector,
	})
	var lastIPs []string
	var lastPort int32
	if err != nil {
		log.Error(err, "failed to watch egress service", "egress", egress)
	} else {
		if last := c.cacheEgressWatchCancel[ci.Name]; last != nil {
			last()
		}
		c.cacheEgressWatchCancel[ci.Name] = cancel
		go func() {
			defer watch.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case event, ok := <-watch.ResultChan():
					if !ok {
						return
					}
					svc := event.Object.(*corev1.Endpoints)
					log.Info("watch egress service", "event", event.Type, "endpoint", utils.KObj(svc))
					ips, err := getIPs(ctx, clientset, egress)
					if err != nil {
						backoff := time.Second
						for {
							time.Sleep(backoff)
							ips, err = getIPs(ctx, clientset, egress)
							if err == nil {
								break
							}
							backoff <<= 1
							if backoff > 16*time.Second {
								backoff = 16 * time.Second
							}
							log.Error(err, "Get IPs for egressIPs")
						}
					}
					port, err := getPort(ctx, clientset, egress)
					if err != nil {
						log.Error(err, "Get port for egressPort")
						continue
					}

					if !reflect.DeepEqual(lastIPs, ips) || lastPort != port {
						lastIPs = ips
						lastPort = port
						c.syncFunc(ctx, ci.Name)
					}
				}
			}
		}()
	}
}

func (c *clusterInformationController) OnAdd(obj interface{}) {
	f := obj.(*v1alpha1.ClusterInformation)
	f = f.DeepCopy()
	c.logger.Info("OnAdd",
		"ClusterInformation", utils.KObj(f),
	)

	c.mut.Lock()
	defer c.mut.Unlock()

	clientset, err := client.NewClientsetFromKubeconfig(f.Spec.Kubeconfig)
	if err != nil {
		c.logger.Error(err, "NewClientsetFromKubeconfig")
	} else {
		c.cacheClientset[f.Name] = clientset
	}

	c.setupWatchEgress(c.ctx, f)
	c.cacheClusterInformation[f.Name] = f
	c.cacheTunnelPorts[f.Name] = newTunnelPorts()

	clusterService := newClusterServiceCache(clusterServiceCacheConfig{
		Clientset: clientset,
		Logger:    c.logger.WithName("service"),
	})
	c.cacheService[f.Name] = clusterService

	err = clusterService.Start(c.ctx)
	if err != nil {
		c.logger.Error(err, "failed start cluster service cache")
	}

	c.syncFunc(c.ctx, f.Name)
}

func (c *clusterInformationController) OnUpdate(oldObj, newObj interface{}) {
	f := newObj.(*v1alpha1.ClusterInformation)
	f = f.DeepCopy()
	c.logger.Info("OnUpdate",
		"ClusterInformation", utils.KObj(f),
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

	c.setupWatchEgress(c.ctx, f)
	c.cacheClusterInformation[f.Name] = f

	c.syncFunc(c.ctx, f.Name)
}

func (c *clusterInformationController) OnDelete(obj interface{}) {
	f := obj.(*v1alpha1.ClusterInformation)
	c.logger.Info("OnDelete",
		"ClusterInformation", utils.KObj(f),
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

	c.syncFunc(c.ctx, f.Name)
}

func (c *clusterInformationController) Get(name string) *v1alpha1.ClusterInformation {
	c.mut.RLock()
	defer c.mut.RUnlock()
	return c.cacheClusterInformation[name]
}

func needWatchEgress(route *v1alpha1.ClusterInformationSpecRoute) bool {
	if route == nil {
		return false
	}
	if route.IP != "" {
		return false
	}
	if route.ServiceNamespace == "" {
		return false
	}
	if route.ServiceName == "" {
		return false
	}
	return true
}

func (c *clusterInformationController) proxy(ctx context.Context, proxy v1alpha1.Proxy) (string, error) {
	if proxy.Proxy != "" {
		return proxy.Proxy, nil
	}
	ci := c.Get(proxy.ClusterName)
	if ci == nil {
		return "", fmt.Errorf("not found cluster %s", proxy.ClusterName)
	}
	if ci.Spec.Ingress == nil {
		return "", fmt.Errorf("not ingress int cluster %s", proxy.ClusterName)
	}

	cli := c.Clientset(proxy.ClusterName)
	if cli == nil {
		return "", fmt.Errorf("not found clientset on cluster %s", proxy.ClusterName)
	}
	ip, err := getIPs(ctx, cli, ci.Spec.Ingress)
	if err != nil {
		return "", err
	}

	port, err := getPort(ctx, cli, ci.Spec.Ingress)
	if err != nil {
		return "", err
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
