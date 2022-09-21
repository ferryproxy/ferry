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

package controllers

import (
	"context"
	"sync"
	"time"

	"github.com/ferryproxy/ferry/pkg/controllers/hub"
	"github.com/ferryproxy/ferry/pkg/controllers/hub/health"
	"github.com/ferryproxy/ferry/pkg/controllers/mcs"
	"github.com/ferryproxy/ferry/pkg/controllers/route"
	"github.com/ferryproxy/ferry/pkg/controllers/route_policy"
	"github.com/ferryproxy/ferry/pkg/utils/trybuffer"
	"github.com/go-logr/logr"
	restclient "k8s.io/client-go/rest"
)

type Controller struct {
	mut                   sync.Mutex
	ctx                   context.Context
	logger                logr.Logger
	config                *restclient.Config
	namespace             string
	hubController         *hub.HubController
	routeController       *route.RouteController
	routePolicyController *route_policy.RoutePolicyController
	mcsController         *mcs.MCSController
	healthController      *health.HealthController
	try                   *trybuffer.TryBuffer
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
	c.try = trybuffer.NewTryBuffer(c.sync, time.Second/10)

	hubController := hub.NewHubController(hub.HubControllerConfig{
		Config:    c.config,
		Namespace: c.namespace,
		Logger:    c.logger.WithName("hub"),
		SyncFunc:  c.try.Try,
	})
	c.hubController = hubController

	routeController := route.NewRouteController(&route.RouteControllerConfig{
		Config:       c.config,
		Namespace:    c.namespace,
		HubInterface: hubController,
		Logger:       c.logger.WithName("route"),
		SyncFunc:     c.try.Try,
	})
	c.routeController = routeController

	routePolicyController := route_policy.NewRoutePolicyController(route_policy.RoutePolicyControllerConfig{
		Config:       c.config,
		Namespace:    c.namespace,
		HubInterface: hubController,
		Logger:       c.logger.WithName("route-policy"),
		SyncFunc:     c.try.Try,
	})
	c.routePolicyController = routePolicyController

	go func() {
		err := hubController.Run(c.ctx)
		if err != nil {
			c.logger.Error(err, "Run HubController")
		}
		cancel()
	}()

	mcsController := mcs.NewMCSController(&mcs.MCSControllerConfig{
		Config:       c.config,
		Namespace:    c.namespace,
		HubInterface: hubController,
		Logger:       c.logger.WithName("mcs"),
	})
	c.mcsController = mcsController
	err := mcsController.Start(ctx)
	if err != nil {
		c.logger.Error(err, "Start MCSController")
	}

	healthController := health.NewHealthController(&health.HealthControllerConfig{
		Config:       c.config,
		HubInterface: hubController,
		Logger:       c.logger.WithName("health"),
	})
	c.healthController = healthController
	err = healthController.Start(ctx)
	if err != nil {
		c.logger.Error(err, "Start MCSController")
	}

	go func() {
		err := routeController.Run(c.ctx)
		if err != nil {
			c.logger.Error(err, "Run RouteController")
		}
		cancel()
	}()

	go func() {
		err := routePolicyController.Run(c.ctx)
		if err != nil {
			c.logger.Error(err, "Run RoutePolicyController")
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

	c.mcsController.Sync(ctx)

	c.routePolicyController.Sync(ctx)

	c.routeController.Sync(ctx)

	c.healthController.Sync(ctx)
}
