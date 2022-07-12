package controller

import (
	"context"
	"sync"
	"time"

	"github.com/ferryproxy/ferry/pkg/ferry-controller/controller/hub"
	"github.com/ferryproxy/ferry/pkg/ferry-controller/controller/route"
	"github.com/ferryproxy/ferry/pkg/ferry-controller/controller/route_policty"
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
	routePolicyController *route_policty.RoutePolicyController
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
	c.try = trybuffer.NewTryBuffer(c.sync, time.Second/2)

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
		ClusterCache: hubController,
		Logger:       c.logger.WithName("route"),
		SyncFunc:     c.try.Try,
	})
	c.routeController = routeController

	routePolicyController := route_policty.NewRoutePolicyController(route_policty.RoutePolicyControllerConfig{
		Config:       c.config,
		Namespace:    c.namespace,
		ClusterCache: hubController,
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
	c.routePolicyController.Sync(ctx)

	c.routeController.Sync(ctx)
}
