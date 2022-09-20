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

	portsclient "github.com/ferryproxy/ferry/pkg/services/ports/client"
	"github.com/ferryproxy/ferry/pkg/services/registry/server"
	"github.com/ferryproxy/ferry/pkg/utils/env"
	"github.com/ferryproxy/ferry/pkg/utils/signals"
	"github.com/go-logr/zapr"
	"github.com/gorilla/handlers"
	"go.uber.org/zap"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	tunnelAddress         = env.GetEnv("TUNNEL_ADDRESS", "")
	portManagerServiceURL = env.GetEnv("PORT_MANAGER_SERVICE_URL", "")
	master                = env.GetEnv("MASTER", "")
	kubeconfig            = env.GetEnv("KUBECONFIG", "")
)

func main() {
	logConfig := zap.NewDevelopmentConfig()
	zapLog, err := logConfig.Build()
	if err != nil {
		os.Exit(1)
	}
	log := zapr.NewLogger(zapLog).WithName("ferry-register")

	config, err := clientcmd.BuildConfigFromFlags(master, kubeconfig)
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

	mux := http.NewServeMux()

	cli := portsclient.NewClient(portManagerServiceURL)
	err = server.Serve(mux, log, config, tunnelAddress, cli.Get)
	if err != nil {
		log.Error(err, "failed to create service router")
		os.Exit(1)
	}
	server := http.Server{
		BaseContext: func(listener net.Listener) context.Context {
			return ctx
		},
		Handler: handlers.LoggingHandler(os.Stderr, mux),
		Addr:    ":8080",
	}
	err = server.ListenAndServe()
	if err != nil {
		log.Error(err, "failed to ListenAndServe")
		os.Exit(1)
	}
}
