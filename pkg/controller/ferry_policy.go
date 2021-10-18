package controller

import (
	"context"
	"fmt"
	"sync"

	"github.com/go-logr/logr"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/kubernetes"

	"github.com/DaoCloud-OpenSource/ferry/api/v1alpha1"
	"github.com/DaoCloud-OpenSource/ferry/pkg/client"
	"github.com/DaoCloud-OpenSource/ferry/pkg/router"
	"github.com/DaoCloud-OpenSource/ferry/pkg/router/original"
	"k8s.io/apimachinery/pkg/labels"
	restclient "k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/cache"
)

type ferryPolicyControllerConfig struct {
	Logger                   logr.Logger
	Config                   *restclient.Config
	Namespace                string
	ClusterInformationGetter ClusterInformationGetter
}

type ferryPolicyController struct {
	clusterInformationGetter ClusterInformationGetter
	ctx                      context.Context
	mut                      sync.RWMutex
	config                   *restclient.Config
	mapping                  map[string]*v1alpha1.FerryPolicy
	mapCancel                map[string]func()
	namespace                string
	logger                   logr.Logger
}

func newFerryPolicyController(conf *ferryPolicyControllerConfig) *ferryPolicyController {
	return &ferryPolicyController{
		clusterInformationGetter: conf.ClusterInformationGetter,
		config:                   conf.Config,
		namespace:                conf.Namespace,
		logger:                   conf.Logger,
		mapping:                  map[string]*v1alpha1.FerryPolicy{},
		mapCancel:                map[string]func(){},
	}
}

func (c *ferryPolicyController) Run(ctx context.Context) error {
	cache, err := cache.New(c.config, cache.Options{
		Namespace: c.namespace,
	})
	if err != nil {
		return err
	}
	informer, err := cache.GetInformer(ctx, &v1alpha1.FerryPolicy{})
	if err != nil {
		return err
	}
	informer.AddEventHandler(c)
	c.ctx = ctx
	return cache.Start(ctx)
}

func (c *ferryPolicyController) OnAdd(obj interface{}) {
	c.mut.Lock()
	defer c.mut.Unlock()

	f := obj.(*v1alpha1.FerryPolicy)
	f = f.DeepCopy()
	c.logger.Info("OnAdd",
		"FerryPolicy", f.Name,
	)

	c.mapping[f.Name] = f

	ctx, cancel := context.WithCancel(c.ctx)
	c.mapCancel[f.Name] = cancel
	go c.sync(ctx, f)
}

func (c *ferryPolicyController) OnUpdate(oldObj, newObj interface{}) {
	c.mut.Lock()
	defer c.mut.Unlock()

	f := newObj.(*v1alpha1.FerryPolicy)
	f = f.DeepCopy()
	c.logger.Info("OnUpdate",
		"FerryPolicy", f.Name,
	)

	c.mapping[f.Name] = f

	cancel, ok := c.mapCancel[f.Name]
	if ok && cancel != nil {
		cancel()
	}

	ctx, cancel := context.WithCancel(c.ctx)
	c.mapCancel[f.Name] = cancel
	go c.sync(ctx, f)
}

func (c *ferryPolicyController) OnDelete(obj interface{}) {
	c.mut.Lock()
	defer c.mut.Unlock()

	f := obj.(*v1alpha1.FerryPolicy)
	c.logger.Info("OnDelete",
		"FerryPolicy", f.Name,
	)

	delete(c.mapping, f.Name)

	cancel, ok := c.mapCancel[f.Name]
	if ok && cancel != nil {
		cancel()
	}

	delete(c.mapCancel, f.Name)
}

func (c *ferryPolicyController) sync(ctx context.Context, policy *v1alpha1.FerryPolicy) {
	c.mut.RLock()
	defer c.mut.RUnlock()

	for _, rule := range policy.Spec.Rules {
		for _, export := range rule.Exports {
			if export.ClusterName == "" {
				continue
			}

			exportCluster := c.clusterInformationGetter.Get(export.ClusterName)
			if exportCluster == nil {
				c.logger.Info("Not found ClusterInformation",
					"FerryPolicy", policy.Name,
					"ClusterInformation", export.ClusterName,
					"Namespace", policy.Namespace,
				)
				continue
			}
			if exportCluster.Spec.Ingress == nil {
				c.logger.Info("Tried to export Service but Ingress is empty",
					"FerryPolicy", policy.Name,
					"ClusterInformation", export.ClusterName,
					"Namespace", policy.Namespace,
				)
				continue
			}

			for _, impor := range rule.Imports {
				if impor.ClusterName == "" || impor.ClusterName == export.ClusterName {
					continue
				}

				importCluster := c.clusterInformationGetter.Get(impor.ClusterName)
				if importCluster == nil {
					c.logger.Info("Not found ClusterInformation",
						"FerryPolicy", policy.Name,
						"ClusterInformation", impor.ClusterName,
						"Namespace", policy.Namespace,
					)
					continue
				}
				if importCluster.Spec.Egress == nil {
					c.logger.Info("Tried to export Service but Egress is empty",
						"FerryPolicy", policy.Name,
						"ClusterInformation", importCluster.Name,
						"Namespace", policy.Namespace,
					)
					continue
				}

				if export.Match == nil {
					export.Match = &v1alpha1.Match{}
				}
				if impor.Match == nil {
					impor.Match = &v1alpha1.Match{}
				}
				if export.Match.Namespace != "" && export.Match.Namespace != impor.Match.Namespace {
					continue
				}

				var matchSet labels.Set
				var err error
				switch {
				case len(export.Match.Labels) != 0 && len(impor.Match.Labels) != 0:
					matchSet, err = mergeMaps(export.Match.Labels, impor.Match.Labels)
					if err != nil {
						c.logger.Error(err, "",
							"FerryPolicy", policy.Name,
							"ClusterInformation", importCluster.Name,
							"Namespace", policy.Namespace,
						)
						continue
					}
				case len(export.Match.Labels) != 0 && len(impor.Match.Labels) == 0:
					matchSet = export.Match.Labels
				case len(export.Match.Labels) == 0 && len(impor.Match.Labels) != 0:
					matchSet = impor.Match.Labels
				}

				exportClientset, err := client.NewClientsetFromKubeconfig(exportCluster.Spec.Kubeconfig)
				if err != nil {
					c.logger.Error(err, "Get Clientset",
						"FerryPolicy", policy.Name,
						"ClusterInformation", exportCluster.Name,
						"Namespace", policy.Namespace,
					)
					continue
				}
				importClientset, err := client.NewClientsetFromKubeconfig(importCluster.Spec.Kubeconfig)
				if err != nil {
					c.logger.Error(err, "Get Clientset",
						"FerryPolicy", policy.Name,
						"ClusterInformation", importCluster.Name,
						"Namespace", policy.Namespace,
					)
					continue
				}

				egressIPs, err := getIPs(ctx, importClientset, importCluster.Spec.Egress)
				if err != nil {
					c.logger.Error(err, "Get IPs",
						"FerryPolicy", policy.Name,
						"ClusterInformation", importCluster.Name,
						"Namespace", policy.Namespace,
					)
					continue
				}

				ingressIPs, err := getIPs(ctx, exportClientset, exportCluster.Spec.Ingress)
				if err != nil {
					c.logger.Error(err, "Get IPs",
						"FerryPolicy", policy.Name,
						"ClusterInformation", exportCluster.Name,
						"Namespace", policy.Namespace,
					)
					continue
				}
				c.logger.Info("Run DataPlaneController",
					"ExportClusterName", exportCluster.Name,
					"ImportClusterName", importCluster.Name,
					"Selector", labels.SelectorFromSet(matchSet),
					"EgressIPs", egressIPs,
					"EgressPort", importCluster.Spec.Egress.Port,
					"IngressIPs", ingressIPs,
					"IngressPort", exportCluster.Spec.Ingress.Port,
				)
				c := NewDataPlaneController(DataPlaneControllerConfig{
					ExportClusterName:          exportCluster.Name,
					ImportClusterName:          importCluster.Name,
					Selector:                   labels.SelectorFromSet(matchSet),
					ExportClientset:            exportClientset,
					ImportClientset:            importClientset,
					Logger:                     c.logger,
					SourceResourceBuilder:      router.ResourceBuilders{original.IngressBuilder{}},
					DestinationResourceBuilder: router.ResourceBuilders{original.EgressBuilder{}, original.ServiceEgressDiscoveryBuilder{}},
					Proxy: router.Proxy{
						RemotePrefix: "ferry",
						EgressIPs:    egressIPs,
						EgressPort:   importCluster.Spec.Egress.Port,
						IngressIPs:   ingressIPs,
						IngressPort:  exportCluster.Spec.Ingress.Port,
						Labels: map[string]string{
							"manage-by": "ferry",
						},
					},
				})
				go c.Run(ctx)
			}
		}
	}
	return
}

func getIPs(ctx context.Context, clientset *kubernetes.Clientset, route *v1alpha1.ClusterInformationSpecRoute) ([]string, error) {
	if route.IP != nil {
		return []string{*route.IP}, nil
	}
	if route.ServiceNamespace == nil {
		return nil, fmt.Errorf("ServiceNamespace is empty")
	}
	if route.ServiceName == nil {
		return nil, fmt.Errorf("ServiceName is empty")
	}
	ep, err := clientset.CoreV1().Endpoints(*route.ServiceNamespace).Get(ctx, *route.ServiceName, metav1.GetOptions{})
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

func mergeMaps(ms ...map[string]string) (map[string]string, error) {
	n := map[string]string{}
	for _, m := range ms {
		for k, v := range m {
			o, ok := n[k]
			if ok && o != v {
				return nil, fmt.Errorf("import and export have different matching values with the same key value %s", k)
			}
			n[k] = v
		}
	}
	return n, nil
}
