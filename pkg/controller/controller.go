package controller

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ferry-proxy/api/apis/ferry/v1alpha1"
	"github.com/ferry-proxy/ferry/pkg/router"
	original "github.com/ferry-proxy/ferry/pkg/router/tunnel"
	"github.com/ferry-proxy/ferry/pkg/utils"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/labels"
	restclient "k8s.io/client-go/rest"
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
	labels                       map[string]string
	updateAllCh                  chan struct{}
}

func NewController(ctx context.Context, config *restclient.Config, namespace string) (*Controller, error) {
	log, err := logr.FromContext(ctx)
	if err != nil {
		return nil, err
	}
	return &Controller{
		logger:                   log,
		config:                   config,
		namespace:                namespace,
		cacheDataPlaneController: map[string]*DataPlaneController{},
		cacheDataPlaneCancel:     map[string]func(){},
		labels: map[string]string{
			"ferry.zsm.io/managed-by": "ferry-controller",
		},
		updateAllCh: make(chan struct{}, 1),
	}, nil
}

func (c *Controller) Run(ctx context.Context) error {
	go func() {
		for range c.updateAllCh {
		next:
			for {
				select {
				case <-c.updateAllCh:
					continue
				case <-time.After(2 * time.Second):
					break next
				case <-ctx.Done():
					return
				}
			}
			list := c.ferryPolicyController.List()
			c.sync(ctx, list, "")
		}
	}()

	clusterInformation := newClusterInformationController(&clusterInformationControllerConfig{
		Config:    c.config,
		Namespace: c.namespace,
		Logger:    c.logger.WithName("cluster-information"),
		SyncFunc: func(ctx context.Context, s string) {
			c.updateAllCh <- struct{}{}
		},
	})
	c.clusterInformationController = clusterInformation
	ferryPolicy := newFerryPolicyController(&ferryPolicyControllerConfig{
		Config:    c.config,
		Namespace: c.namespace,
		Logger:    c.logger.WithName("ferry-policy"),
		SyncFunc: func(ctx context.Context, policy *v1alpha1.FerryPolicy) {
			c.updateAllCh <- struct{}{}
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

	// TODO remove this
	time.Sleep(time.Second * 2)

	go func() {
		err := ferryPolicy.Run(ctx)
		if err != nil {
			c.logger.Error(err, "Run FerryPolicyController")
		}
		cancel()
	}()

	<-ctx.Done()
	return nil
}

func (c *Controller) sync(ctx context.Context, policies []*v1alpha1.FerryPolicy, syncCluster string) {
	c.mut.Lock()
	defer c.mut.Unlock()

	updated := map[string]struct{}{}

	for _, policy := range policies {
		for _, rule := range policy.Spec.Rules {
			for _, export := range rule.Exports {
				if export.ClusterName == "" {
					continue
				}

				log := c.logger.WithValues(
					"FerryPolicy", utils.KObj(policy),
				)

				log = log.WithValues(
					"ExportClusterName", export.ClusterName,
				)
				exportCluster := c.clusterInformationController.Get(export.ClusterName)
				if exportCluster == nil {
					log.Info("Not found ClusterInformation")
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

					log = log.WithValues(
						"ImportClusterName", impor.ClusterName,
					)
					importCluster := c.clusterInformationController.Get(impor.ClusterName)
					if importCluster == nil {
						c.logger.Info("Not found ClusterInformation")
						continue
					}
					if importCluster.Spec.Egress == nil {
						c.logger.Info("Tried to import Service but Egress is empty")
						continue
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
							c.logger.Error(err, "")
							continue
						}
					case len(export.Match.Labels) != 0 && len(impor.Match.Labels) == 0:
						matchSet = export.Match.Labels
					case len(export.Match.Labels) == 0 && len(impor.Match.Labels) != 0:
						matchSet = impor.Match.Labels
					}

					exportClientset := c.clusterInformationController.Clientset(exportCluster.Name)
					if exportClientset == nil {
						c.logger.Error(fmt.Errorf("not found %q", exportCluster.Name), "Get Clientset")
						continue
					}
					importClientset := c.clusterInformationController.Clientset(importCluster.Name)
					if importClientset == nil {
						c.logger.Error(fmt.Errorf("not found %q", importCluster.Name), "Get Clientset")
						continue
					}

					inClusterEgressIPs, err := getIPs(ctx, importClientset, importCluster.Spec.Egress)
					if err != nil {
						c.logger.Error(err, "Get IPs for inClusterEgressIPs")
						continue
					}

					exportIngressIPs, err := getIPs(ctx, exportClientset, exportCluster.Spec.Ingress)
					if err != nil {
						c.logger.Error(err, "Get IPs for exportIngressIPs")
						continue
					}
					exportIngressPort, err := getPort(ctx, exportClientset, exportCluster.Spec.Ingress)
					if err != nil {
						c.logger.Error(err, "Get port for exportIngressPort")
						continue
					}

					importIngressIPs, err := getIPs(ctx, importClientset, importCluster.Spec.Ingress)
					if err != nil {
						c.logger.Error(err, "Get IPs for importIngressIPs")
						continue
					}
					importIngressPort, err := getPort(ctx, importClientset, importCluster.Spec.Ingress)
					if err != nil {
						c.logger.Error(err, "Get port for importIngressPort")
						continue
					}

					reverse := false

					if len(exportIngressIPs) == 0 {
						if len(importIngressIPs) == 0 {
							c.logger.Info("Tried to export Service but Ingress is empty")
							continue
						} else {
							reverse = true
						}
					}

					key := fmt.Sprintf("%s-%#v|%s-%#v", export.ClusterName, export.Match, impor.ClusterName, impor.Match)

					var exportPortOffset int32 = 40000
					var importPortOffset int32 = 50000
					if reverse {
						exportPortOffset = 45000
						exportPortOffset = 55000
					}

					log := log.WithName("data-plane").
						WithValues(
							"Selector", labels.SelectorFromSet(matchSet),
							"InClusterEgressIPs", inClusterEgressIPs,
							"ExportIngressIPs", exportIngressIPs,
							"ExportIngressPort", exportIngressPort,
							"ImportIngressIPs", importIngressIPs,
							"ImportIngressPort", importIngressPort,
							"Reverse", reverse,
						)
					log.Info("Run DataPlaneController")
					proxy := router.Proxy{
						Labels: utils.MergeMap(c.labels, map[string]string{
							"exported-from": exportCluster.Name,
							"imported-to":   importCluster.Name,
						}),
						RemotePrefix:     "ferry",
						TunnelNamespace:  "ferry-tunnel-system",
						ExportPortOffset: exportPortOffset,
						ImportPortOffset: importPortOffset,
						Reverse:          reverse,

						ExportClusterName: exportCluster.Name,
						ImportClusterName: importCluster.Name,

						InClusterEgressIPs: inClusterEgressIPs,

						ExportIngressIPs:  exportIngressIPs,
						ExportIngressPort: exportIngressPort,

						ImportIngressIPs:  importIngressIPs,
						ImportIngressPort: importIngressPort,
					}
					dataPlane := NewDataPlaneController(DataPlaneControllerConfig{
						ExportCluster:              exportCluster,
						ImportCluster:              importCluster,
						Selector:                   labels.SelectorFromSet(matchSet),
						ExportClientset:            exportClientset,
						ImportClientset:            importClientset,
						Logger:                     log,
						SourceResourceBuilder:      router.ResourceBuilders{original.IngressBuilder},
						DestinationResourceBuilder: router.ResourceBuilders{original.EgressBuilder, original.ServiceEgressDiscoveryBuilder},
						Proxy:                      proxy,
					})
					if cancel, ok := c.cacheDataPlaneCancel[key]; ok {
						cancel()
					}
					c.cacheDataPlaneController[key] = dataPlane
					ctx, cancel := context.WithCancel(ctx)
					c.cacheDataPlaneCancel[key] = cancel

					err = dataPlane.Start(ctx)
					if err != nil {
						c.logger.Error(err, "Start Data Plane")
					}

					updated[key] = struct{}{}
				}
			}
		}
	}

	for key := range c.cacheDataPlaneCancel {
		_, ok := updated[key]
		if !ok {
			c.cacheDataPlaneController[key].Cleanup(ctx)
			delete(c.cacheDataPlaneController, key)
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
