package controller

import (
	"context"
	"sync"
	"time"

	"github.com/ferry-proxy/ferry/pkg/controller/cluster_information"
	"github.com/ferry-proxy/ferry/pkg/controller/ferry_policty"
	"github.com/ferry-proxy/ferry/pkg/controller/mapping_rule"
	"github.com/ferry-proxy/utils/trybuffer"
	"github.com/go-logr/logr"
	restclient "k8s.io/client-go/rest"
)

type Controller struct {
	mut                          sync.Mutex
	ctx                          context.Context
	logger                       logr.Logger
	config                       *restclient.Config
	namespace                    string
	clusterInformationController *cluster_information.ClusterInformationController
	mappingRuleController        *mapping_rule.MappingRuleController
	ferryPolicyController        *ferry_policty.FerryPolicyController
	try                          *trybuffer.TryBuffer
}

type ControllerConfig struct {
	Config    *restclient.Config
	Logger    logr.Logger
	Namespace string
}

func NewController(conf *ControllerConfig) *Controller {
	return &Controller{
		logger:    conf.Logger,
		config:    conf.Config,
		namespace: conf.Namespace,
	}
}

func (c *Controller) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	c.ctx = ctx
	c.try = trybuffer.NewTryBuffer(c.sync, time.Second/2)

	clusterInformation := cluster_information.NewClusterInformationController(cluster_information.ClusterInformationControllerConfig{
		Config:    c.config,
		Namespace: c.namespace,
		Logger:    c.logger.WithName("cluster-information"),
		SyncFunc:  c.try.Try,
	})
	c.clusterInformationController = clusterInformation

	mappingRule := mapping_rule.NewMappingRuleController(&mapping_rule.MappingRuleControllerConfig{
		Config:                       c.config,
		Namespace:                    c.namespace,
		ClusterInformationController: clusterInformation,
		Logger:                       c.logger.WithName("mapping-rule"),
		SyncFunc:                     c.try.Try,
	})
	c.mappingRuleController = mappingRule

	ferryPolicy := ferry_policty.NewFerryPolicyController(ferry_policty.FerryPolicyControllerConfig{
		Config:                       c.config,
		Namespace:                    c.namespace,
		ClusterInformationController: clusterInformation,
		Logger:                       c.logger.WithName("ferry-policy"),
		SyncFunc:                     c.try.Try,
	})
	c.ferryPolicyController = ferryPolicy

	go func() {
		err := clusterInformation.Run(c.ctx)
		if err != nil {
			c.logger.Error(err, "Run ClusterInformationController")
		}
		cancel()
	}()

	go func() {
		err := mappingRule.Run(c.ctx)
		if err != nil {
			c.logger.Error(err, "Run MappingRuleController")
		}
		cancel()
	}()

	go func() {
		err := ferryPolicy.Run(c.ctx)
		if err != nil {
			c.logger.Error(err, "Run FerryPolicyController")
		}
		cancel()
	}()

	select {
	case <-c.ctx.Done():
		c.try.Close()
		return c.ctx.Err()
	case <-time.After(5 * time.Second):
		c.try.Try()
	}

	for {
		select {
		case <-c.ctx.Done():
			c.try.Close()
			return c.ctx.Err()
		case <-time.After(time.Minute):
			c.try.Try()
		}
	}
}

func (c *Controller) sync() {
	c.mut.Lock()
	defer c.mut.Unlock()

	ctx := c.ctx
	c.ferryPolicyController.Sync(ctx)

	c.mappingRuleController.Sync(ctx)
}
