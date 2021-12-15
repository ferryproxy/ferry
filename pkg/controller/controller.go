package controller

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-logr/logr"

	"github.com/ferry-proxy/ferry/api/v1alpha1"
	"github.com/ferry-proxy/ferry/pkg/client"
	"github.com/ferry-proxy/ferry/pkg/router"
	"github.com/ferry-proxy/ferry/pkg/router/original"
	"k8s.io/apimachinery/pkg/labels"

	restclient "k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type Controller struct {
	mut                          sync.Mutex
	logger                       logr.Logger
	config                       *restclient.Config
	namespace                    string
	clusterInformationController *clusterInformationController
	ferryPolicyController        *ferryPolicyController
	cacheDataPlaneController     map[string]*DataPlaneController
	cacheDataPlaneCancel         map[string]func()
}

func NewController(ctx context.Context, config *restclient.Config, namespace string) (*Controller, error) {
	return &Controller{
		logger:                   log.FromContext(ctx),
		config:                   config,
		namespace:                namespace,
		cacheDataPlaneController: map[string]*DataPlaneController{},
		cacheDataPlaneCancel:     map[string]func(){},
	}, nil
}

func (c *Controller) Start(ctx context.Context) error {
	clusterInformation := newClusterInformationController(&clusterInformationControllerConfig{
		Config:    c.config,
		Namespace: c.namespace,
		Logger:    c.logger.WithName("cluster-infomation"),
		SyncFunc: func(ctx context.Context, s string) {
			go func() {
				list := c.ferryPolicyController.List()
				for _, item := range list {
					c.sync(ctx, item, s)
				}
			}()
		},
	})
	c.clusterInformationController = clusterInformation
	ferryPolicy := newFerryPolicyController(&ferryPolicyControllerConfig{
		Config:    c.config,
		Namespace: c.namespace,
		Logger:    c.logger.WithName("ferry-policy"),
		SyncFunc: func(ctx context.Context, policy *v1alpha1.FerryPolicy) {
			go c.sync(ctx, policy.Name, "")
		},
	})
	c.ferryPolicyController = ferryPolicy

	ctx, cancel := context.WithCancel(ctx)
	go func() {
		err := clusterInformation.Run(ctx)
		if err != nil {
			c.logger.Error(err, "Run ClusterInformationController")
		}
		cancel()
	}()

	time.Sleep(time.Second * 2)

	go func() {
		err := ferryPolicy.Run(ctx)
		if err != nil {
			c.logger.Error(err, "Run FerryPolicyController")
		}
		cancel()
	}()
	return nil
}

func (c *Controller) sync(ctx context.Context, policyName string, syncCluster string) {
	c.mut.Lock()
	defer c.mut.Unlock()
	policy := c.ferryPolicyController.Get(policyName)
	if policy == nil {
		return
	}
	for _, rule := range policy.Spec.Rules {
		for _, export := range rule.Exports {
			if export.ClusterName == "" {
				continue
			}

			exportCluster := c.clusterInformationController.Get(export.ClusterName)
			if exportCluster == nil {
				c.logger.Info("Not found ClusterInformation",
					"FerryPolicy", uniqueKey(policy.Name, policy.Namespace),
					"ClusterInformation", export.ClusterName,
				)
				continue
			}
			if exportCluster.Spec.Ingress == nil {
				c.logger.Info("Tried to export Service but Ingress is empty",
					"FerryPolicy", uniqueKey(policy.Name, policy.Namespace),
					"ClusterInformation", export.ClusterName,
				)
				continue
			}

			for _, impor := range rule.Imports {
				if impor.ClusterName == "" || impor.ClusterName == export.ClusterName {
					continue
				}

				if syncCluster != "" &&
					impor.ClusterName != syncCluster && export.ClusterName != syncCluster {
					continue
				}

				importCluster := c.clusterInformationController.Get(impor.ClusterName)
				if importCluster == nil {
					c.logger.Info("Not found ClusterInformation",
						"FerryPolicy", uniqueKey(policy.Name, policy.Namespace),
						"ClusterInformation", impor.ClusterName,
					)
					continue
				}
				if importCluster.Spec.Egress == nil {
					c.logger.Info("Tried to export Service but Egress is empty",
						"FerryPolicy", uniqueKey(policy.Name, policy.Namespace),
						"ClusterInformation", importCluster.Name,
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
							"FerryPolicy", uniqueKey(policy.Name, policy.Namespace),
							"ClusterInformation", importCluster.Name,
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
						"FerryPolicy", uniqueKey(policy.Name, policy.Namespace),
						"ClusterInformation", exportCluster.Name,
					)
					continue
				}
				importClientset, err := client.NewClientsetFromKubeconfig(importCluster.Spec.Kubeconfig)
				if err != nil {
					c.logger.Error(err, "Get Clientset",
						"FerryPolicy", uniqueKey(policy.Name, policy.Namespace),
						"ClusterInformation", importCluster.Name,
					)
					continue
				}

				egressIPs, err := getIPs(ctx, importClientset, importCluster.Spec.Egress)
				if err != nil {
					c.logger.Error(err, "Get IPs",
						"FerryPolicy", uniqueKey(policy.Name, policy.Namespace),
						"ClusterInformation", importCluster.Name,
					)
					continue
				}

				ingressIPs, err := getIPs(ctx, exportClientset, exportCluster.Spec.Ingress)
				if err != nil {
					c.logger.Error(err, "Get IPs",
						"FerryPolicy", uniqueKey(policy.Name, policy.Namespace),
						"ClusterInformation", exportCluster.Name,
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

				key := exportCluster.Name + "|" + importCluster.Name
				dataPlane := NewDataPlaneController(DataPlaneControllerConfig{
					ExportClusterName: exportCluster.Name,
					ImportClusterName: importCluster.Name,
					Selector:          labels.SelectorFromSet(matchSet),
					ExportClientset:   exportClientset,
					ImportClientset:   importClientset,
					Logger: c.logger.WithName("data-plane").
						WithValues("ExportCluster", exportCluster.Name).
						WithValues("ImportCluster", importCluster.Name),
					SourceResourceBuilder:      router.ResourceBuilders{original.IngressBuilder{}},
					DestinationResourceBuilder: router.ResourceBuilders{original.EgressBuilder{}, original.ServiceEgressDiscoveryBuilder{}},
					Proxy: router.Proxy{
						RemotePrefix: "ferry",
						EgressIPs:    egressIPs,
						EgressPort:   importCluster.Spec.Egress.Port,
						IngressIPs:   ingressIPs,
						IngressPort:  exportCluster.Spec.Ingress.Port,
						Labels: map[string]string{
							"managed-by": "ferry",
						},
					},
				})
				if cancel, ok := c.cacheDataPlaneCancel[key]; ok {
					cancel()
				}
				c.cacheDataPlaneController[key] = dataPlane
				ctx, cancel := context.WithCancel(ctx)
				c.cacheDataPlaneCancel[key] = cancel

				go dataPlane.Run(ctx)
			}
		}
	}
	return
}

func uniqueKey(name, ns string) string {
	return name + "." + ns
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
