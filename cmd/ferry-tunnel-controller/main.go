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
	"net"
	"net/http"
	"os"

	"github.com/ferryproxy/ferry/pkg/client"
	"github.com/ferryproxy/ferry/pkg/consts"
	healthserver "github.com/ferryproxy/ferry/pkg/services/health/server"
	portsserver "github.com/ferryproxy/ferry/pkg/services/ports/server"
	"github.com/ferryproxy/ferry/pkg/tunnel/controllers"
	"github.com/ferryproxy/ferry/pkg/utils/env"
	"github.com/ferryproxy/ferry/pkg/utils/signals"
	"github.com/go-logr/zapr"
	"github.com/gorilla/handlers"
	"go.uber.org/zap"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	serviceName    = env.GetEnv("SERVICE_NAME", consts.FerryTunnelName)
	serviceAddress = env.GetEnv("SERVICE_ADDRESS", "")
	namespace      = env.GetEnv("NAMESPACE", consts.FerryTunnelNamespace)
	master         = env.GetEnv("MASTER", "")
	kubeconfig     = env.GetEnv("KUBECONFIG", "")
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

	clientset, err := client.NewForConfig(config)
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
		svcSyncer := controllers.NewDiscoveryController(&controllers.DiscoveryControllerConfig{
			Clientset:     clientset,
			Logger:        log.WithName("discovery-controller"),
			Namespace:     namespace,
			LabelSelector: consts.TunnelConfigKey + "=" + consts.TunnelConfigDiscoverValue,
		})

		epWatcher := controllers.NewEndpointWatcher(&controllers.EndpointWatcherConfig{
			Clientset: clientset,
			Name:      serviceName,
			Namespace: namespace,
			SyncFunc:  svcSyncer.UpdateIPs,
		})

		authorizedController := controllers.NewAuthorizedController(&controllers.AuthorizedControllerConfig{
			Clientset:     clientset,
			Logger:        log.WithName("authorized-controller"),
			Namespace:     namespace,
			LabelSelector: consts.TunnelConfigKey + "=" + consts.TunnelConfigAuthorizedValue,
		})

		allowController := controllers.NewAllowController(&controllers.AllowControllerConfig{
			Clientset:     clientset,
			Logger:        log.WithName("allow-controller"),
			Namespace:     namespace,
			LabelSelector: consts.TunnelConfigKey + "=" + consts.TunnelConfigAllowValue,
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

		go func() {
			err := authorizedController.Run(ctx)
			if err != nil {
				log.Error(err, "failed to run authorized controller")
			}
		}()

		go func() {
			err := allowController.Run(ctx)
			if err != nil {
				log.Error(err, "failed to run allow controller")
			}
		}()
	}

	if serviceAddress != "" {
		go func() {
			mux := http.NewServeMux()

			err = portsserver.Serve(mux, log)
			if err != nil {
				log.Error(err, "failed to create ports serve")
				os.Exit(1)
			}

			err = healthserver.Serve(mux, log)
			if err != nil {
				log.Error(err, "failed to create health serve")
				os.Exit(1)
			}
			server := http.Server{
				BaseContext: func(listener net.Listener) context.Context {
					return ctx
				},
				Handler: handlers.LoggingHandler(os.Stderr, mux),
				Addr:    serviceAddress,
			}
			err = server.ListenAndServe()
			if err != nil {
				log.Error(err, "failed to ListenAndServe")
				os.Exit(1)
			}
		}()
	}

	ctr := controllers.NewRuntimeController(&controllers.RuntimeControllerConfig{
		Namespace:     namespace,
		LabelSelector: consts.TunnelConfigKey + "=" + consts.TunnelConfigRulesValue,
		Clientset:     clientset,
		Logger:        log.WithName("runtime-controller"),
	})

	err = ctr.Run(ctx)
	if err != nil {
		log.Error(err, "failed to run runtime controller")
	}
}
