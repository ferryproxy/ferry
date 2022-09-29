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

	"github.com/ferryproxy/ferry/pkg/client"
	"github.com/ferryproxy/ferry/pkg/controllers/hub"
	"github.com/ferryproxy/ferry/pkg/controllers/mcs"
	"github.com/ferryproxy/ferry/pkg/controllers/route"
	"github.com/ferryproxy/ferry/pkg/controllers/route_policy"
	"github.com/ferryproxy/ferry/pkg/utils/trybuffer"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/util/wait"
)

type Controller struct {
	mut                   sync.Mutex
	ctx                   context.Context
	logger                logr.Logger
	clientset             client.Interface
	namespace             string
	hubController         *hub.HubController
	routeController       *route.RouteController
	routePolicyController *route_policy.RoutePolicyController
	mcsController         *mcs.MCSController
	try                   *trybuffer.TryBuffer
}

type ControllerConfig struct {
	Clientset client.Interface
	Logger    logr.Logger
	Namespace string
}

func NewController(conf *ControllerConfig) *Controller {
	return &Controller{
		logger:    conf.Logger,
		clientset: conf.Clientset,
		namespace: conf.Namespace,
	}
}

func (c *Controller) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	c.ctx = ctx
	c.try = trybuffer.NewTryBuffer(c.sync, time.Second)

	hubController := hub.NewHubController(hub.HubControllerConfig{
		Clientset: c.clientset,
		Namespace: c.namespace,
		Logger:    c.logger.WithName("hub"),
		SyncFunc:  c.try.Try,
	})
	c.hubController = hubController

	routeController := route.NewRouteController(&route.RouteControllerConfig{
		Clientset:    c.clientset,
		Namespace:    c.namespace,
		HubInterface: hubController,
		Logger:       c.logger.WithName("route"),
		SyncFunc:     c.try.Try,
	})
	c.routeController = routeController

	routePolicyController := route_policy.NewRoutePolicyController(route_policy.RoutePolicyControllerConfig{
		Clientset:    c.clientset,
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
		Clientset:    c.clientset,
		Namespace:    c.namespace,
		HubInterface: hubController,
		Logger:       c.logger.WithName("mcs"),
	})
	c.mcsController = mcsController
	err := mcsController.Start(ctx)
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

	wait.Until(c.sync, time.Minute, c.ctx.Done())
	c.try.Close()
	return c.ctx.Err()
}

func (c *Controller) sync() {
	c.mut.Lock()
	defer c.mut.Unlock()

	ctx := c.ctx

	c.hubController.Sync(ctx)

	c.mcsController.Sync(ctx)

	c.routePolicyController.Sync(ctx)

	c.routeController.Sync(ctx)
}
