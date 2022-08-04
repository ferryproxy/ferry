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

package main

import (
	"context"
	"os"

	"github.com/go-logr/zapr"
	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/ferryproxy/ferry/pkg/consts"
	"github.com/ferryproxy/ferry/pkg/ferry-tunnel/controller"
	"github.com/ferryproxy/ferry/pkg/utils/env"
	"github.com/ferryproxy/ferry/pkg/utils/signals"
)

var (
	serviceName   = env.GetEnv("SERVICE_NAME", consts.FerryTunnelName)
	namespace     = env.GetEnv("NAMESPACE", consts.FerryTunnelNamespace)
	labelSelector = env.GetEnv("LABEL_SELECTOR", "tunnel.ferryproxy.io/service=inject")
	master        = env.GetEnv("MASTER", "")
	kubeconfig    = env.GetEnv("KUBECONFIG", "")
)

const (
	conf = "./bridge.conf"
)

func main() {
	logConfig := zap.NewDevelopmentConfig()
	zapLog, err := logConfig.Build()
	if err != nil {
		os.Exit(1)
	}
	log := zapr.NewLogger(zapLog).WithName("ferry-tunnel-controller")

	config, err := clientcmd.BuildConfigFromFlags(master, kubeconfig)
	if err != nil {
		log.Error(err, "failed to create kubernetes client")
		os.Exit(1)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Error(err, "failed to create kubernetes client")
		os.Exit(1)
	}

	stopCh := signals.SetupNotifySignalHandler()
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		<-stopCh
		cancel()
	}()

	if serviceName != "" {
		svcSyncer := controller.NewServiceSyncer(&controller.ServiceSyncerConfig{
			Clientset:     clientset,
			Logger:        log.WithName("service-syncer"),
			LabelSelector: labelSelector,
		})

		epWatcher := controller.NewEndpointWatcher(&controller.EndpointWatcherConfig{
			Clientset: clientset,
			Name:      serviceName,
			Namespace: namespace,
			SyncFunc:  svcSyncer.UpdateIPs,
		})

		go func() {
			err = epWatcher.Run(ctx)
			if err != nil {
				log.Error(err, "failed to run endpoint watcher")
			}
		}()

		go func() {
			err := svcSyncer.Run(ctx)
			if err != nil {
				log.Error(err, "failed to run service syncer")
			}
		}()
	}

	ctr := controller.NewRuntimeController(&controller.RuntimeControllerConfig{
		Namespace:     namespace,
		LabelSelector: labelSelector,
		Clientset:     clientset,
		Logger:        log.WithName("runtime-controller"),
		Conf:          conf,
	})

	err = ctr.Run(ctx)
	if err != nil {
		log.Error(err, "failed to run runtime controller")
	}
}
