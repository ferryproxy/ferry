package controller

import (
	"context"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/log"

	restclient "k8s.io/client-go/rest"
)

type Controller struct {
	config    *restclient.Config
	namespace string
}

func NewController(config *restclient.Config, namespace string) (*Controller, error) {
	return &Controller{
		config:    config,
		namespace: namespace,
	}, nil
}

func (c *Controller) Start(ctx context.Context) error {
	logger := log.FromContext(ctx)

	clusterInformation := newClusterInformationController(&clusterInformationControllerConfig{
		Config:    c.config,
		Namespace: c.namespace,
		Logger:    logger,
	})
	ferryPolicy := newFerryPolicyController(&ferryPolicyControllerConfig{
		Config:                   c.config,
		Namespace:                c.namespace,
		ClusterInformationGetter: clusterInformation,
		Logger:                   logger,
	})

	ctx, cancel := context.WithCancel(ctx)
	go func() {
		err := clusterInformation.Run(ctx)
		if err != nil {
			logger.Error(err, "Run ClusterInformationController")
		}
		cancel()
	}()

	time.Sleep(time.Second * 2)

	go func() {
		err := ferryPolicy.Run(ctx)
		if err != nil {
			logger.Error(err, "Run FerryPolicyController")
		}
		cancel()
	}()
	return nil
}
