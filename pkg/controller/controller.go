package controller

import (
	"context"
	"encoding/json"
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
	cacheDataPlaneController     map[ClusterPair]*DataPlaneController
	cacheMatchRule               map[string]map[string][]MatchRule
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
		cacheDataPlaneController: map[ClusterPair]*DataPlaneController{},
		cacheMatchRule:           map[string]map[string][]MatchRule{},
		updateAllCh:              make(chan struct{}, 1),
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

type MatchRule struct {
	Export v1alpha1.Match
	Import v1alpha1.Match
}

func CalculateMatchRulePatch(older, newer []MatchRule) (updated, deleted []MatchRule) {
	if len(older) == 0 {
		return newer, nil
	}

	exist := map[string]MatchRule{}

	for _, r := range older {
		data, _ := json.Marshal(r)
		exist[string(data)] = r
	}

	for _, r := range newer {
		data, _ := json.Marshal(r)
		updated = append(updated, r)
		delete(exist, string(data))
	}
	for _, r := range exist {
		deleted = append(deleted, r)
	}
	return updated, deleted
}

type ClusterPair struct {
	Export string
	Import string
}

func CalculateClusterPatch(older, newer map[string]map[string][]MatchRule) (updated, deleted []ClusterPair) {
	exist := map[ClusterPair]struct{}{}

	for exportName, other := range older {
		for importName := range other {
			r := ClusterPair{
				Export: exportName,
				Import: importName,
			}
			exist[r] = struct{}{}
		}
	}

	for exportName, other := range newer {
		for importName := range other {
			r := ClusterPair{
				Export: exportName,
				Import: importName,
			}
			updated = append(updated, r)
			delete(exist, r)
		}
	}

	for r := range exist {
		deleted = append(deleted, r)
	}
	return updated, deleted
}

func (c *Controller) getMatchRules(policies []*v1alpha1.FerryPolicy) map[string]map[string][]MatchRule {
	mapping := map[string]map[string][]MatchRule{}

	for _, policy := range policies {
		for _, rule := range policy.Spec.Rules {
			for _, export := range rule.Exports {
				if export.ClusterName == "" {
					continue
				}
				if _, ok := mapping[export.ClusterName]; !ok {
					mapping[export.ClusterName] = map[string][]MatchRule{}
				}
				for _, impor := range rule.Imports {
					if impor.ClusterName == "" || impor.ClusterName == export.ClusterName {
						continue
					}
					if _, ok := mapping[export.ClusterName][impor.ClusterName]; !ok {
						mapping[export.ClusterName][impor.ClusterName] = []MatchRule{}
					}

					matchRule := MatchRule{
						Export: export.Match,
						Import: impor.Match,
					}
					mapping[export.ClusterName][impor.ClusterName] = append(mapping[export.ClusterName][impor.ClusterName], matchRule)
				}
			}
		}
	}
	return mapping
}

func (c *Controller) sync(ctx context.Context, policies []*v1alpha1.FerryPolicy, syncCluster string) {
	c.mut.Lock()
	defer c.mut.Unlock()

	newerMatchRules := c.getMatchRules(policies)
	defer func() {
		c.cacheMatchRule = newerMatchRules
	}()

	updated, deleted := CalculateClusterPatch(c.cacheMatchRule, newerMatchRules)

	for _, r := range deleted {
		logger := c.logger.WithValues("export", r.Export, "import", r.Import)
		logger.Info("Delete data plane")
		c.cleanupDataPlane(r.Export, r.Import)
	}

	for _, r := range updated {
		logger := c.logger.WithValues("export", r.Export, "import", r.Import)
		logger.Info("Update data plane")
		dataPlane, err := c.startDataPlane(logr.NewContext(ctx, logger), r.Export, r.Import)
		if err != nil {
			logger.Error(err, "start data plane")
			continue
		}
		if newerMatchRules[r.Export] != nil && newerMatchRules[r.Export][r.Import] != nil {
			older := c.cacheMatchRule[r.Export][r.Import]
			newer := newerMatchRules[r.Export][r.Import]
			updated, deleted := CalculateMatchRulePatch(older, newer)
			for _, rule := range updated {
				logger.Info("Update rule", "rule", rule)
				switch {
				case (rule.Import.Name != "" || rule.Export.Name != "") &&
					(len(rule.Export.Labels) == 0 && len(rule.Import.Labels) == 0):
					dataPlane.RegistryObj(
						utils.ObjectRef{Name: rule.Export.Name, Namespace: rule.Export.Namespace},
						utils.ObjectRef{Name: rule.Import.Name, Namespace: rule.Import.Namespace},
					)

				case (len(rule.Export.Labels) != 0 || len(rule.Import.Labels) != 0) &&
					(rule.Import.Name == "" && rule.Export.Name == ""):
					if (rule.Export.Namespace != "" || rule.Import.Namespace != "") &&
						rule.Export.Namespace != rule.Import.Namespace {
						logger.Info("Tried to import Service but Namespace is not matched")
						continue
					}

					matchSet := utils.MergeMap(rule.Export.Labels, rule.Import.Labels)
					dataPlane.RegistrySelector(labels.Set(matchSet).AsSelector())
				}
			}

			for _, rule := range deleted {
				logger.Info("Delete rule", "rule", rule)
				switch {
				case (rule.Import.Name != "" || rule.Export.Name != "") &&
					(len(rule.Export.Labels) == 0 && len(rule.Import.Labels) == 0):
					dataPlane.UnregistryObj(
						utils.ObjectRef{Name: rule.Export.Name, Namespace: rule.Export.Namespace},
						utils.ObjectRef{Name: rule.Import.Name, Namespace: rule.Import.Namespace},
					)

				case (len(rule.Export.Labels) != 0 || len(rule.Import.Labels) != 0) &&
					(rule.Import.Name == "" && rule.Export.Name == ""):
					if (rule.Export.Namespace != "" || rule.Import.Namespace != "") &&
						rule.Export.Namespace != rule.Import.Namespace {
						logger.Info("Tried to import Service but Namespace is not matched")
						continue
					}

					matchSet := utils.MergeMap(rule.Export.Labels, rule.Import.Labels)
					dataPlane.UnregistrySelector(labels.Set(matchSet).AsSelector())
				}
			}

		}
		dataPlane.trySync()
	}

	return
}

func (c *Controller) cleanupDataPlane(exportClusterName, importClusterName string) {
	key := ClusterPair{
		Export: exportClusterName,
		Import: importClusterName,
	}
	dataPlane := c.cacheDataPlaneController[key]
	if dataPlane != nil {
		dataPlane.Cleanup(context.TODO())
		delete(c.cacheDataPlaneController, key)
	}
}

func (c *Controller) startDataPlane(ctx context.Context, exportClusterName, importClusterName string) (*DataPlaneController, error) {
	key := ClusterPair{
		Export: exportClusterName,
		Import: importClusterName,
	}
	dataPlane := c.cacheDataPlaneController[key]
	if dataPlane != nil {
		return dataPlane, nil
	}

	exportClientset := c.clusterInformationController.Clientset(exportClusterName)
	if exportClientset == nil {
		return nil, fmt.Errorf("not found clientset %q", exportClusterName)
	}
	importClientset := c.clusterInformationController.Clientset(importClusterName)
	if importClientset == nil {
		return nil, fmt.Errorf("not found clientset %q", importClusterName)
	}

	exportCluster := c.clusterInformationController.Get(exportClusterName)
	if exportCluster == nil {
		return nil, fmt.Errorf("not found cluster information %q", exportCluster)
	}

	importCluster := c.clusterInformationController.Get(importClusterName)
	if importCluster == nil {
		return nil, fmt.Errorf("not found cluster information %q", importClusterName)
	}

	dataPlane = NewDataPlaneController(DataPlaneControllerConfig{
		ClusterInformationController: c.clusterInformationController,
		ImportClusterName:            importClusterName,
		ExportClusterName:            exportClusterName,
		ExportCluster:                exportCluster,
		ImportCluster:                importCluster,
		ExportClientset:              exportClientset,
		ImportClientset:              importClientset,
		Logger:                       logr.FromContextOrDiscard(ctx),
		SourceResourceBuilder:        router.ResourceBuilders{original.IngressBuilder},
		DestinationResourceBuilder:   router.ResourceBuilders{original.EgressBuilder, original.ServiceEgressDiscoveryBuilder},
	})
	c.cacheDataPlaneController[key] = dataPlane

	err := dataPlane.Start(ctx)
	if err != nil {
		return nil, err
	}
	return dataPlane, nil
}
