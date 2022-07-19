/*
Copyright 2021 FerryProxy Authors.

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
	"syscall"

	"github.com/ferryproxy/ferry/pkg/consts"
	"github.com/ferryproxy/ferry/pkg/ferry-controller/controller"
	"github.com/ferryproxy/ferry/pkg/utils/env"
	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"github.com/wzshiming/notify"
	"go.uber.org/zap"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	ctx, globalCancel = context.WithCancel(context.Background())
	log               = logr.Discard()
	master            = env.GetEnv("MASTER", "")
	kubeconfig        = env.GetEnv("KUBECONFIG", "")
	namespace         = env.GetEnv("NAMESPACE", consts.FerryNamespace)
)

func init() {
	logConfig := zap.NewDevelopmentConfig()
	zapLog, err := logConfig.Build()
	if err != nil {
		os.Exit(1)
	}
	log = zapr.NewLogger(zapLog)

	signals := []os.Signal{syscall.SIGINT, syscall.SIGTERM}
	notify.OnceSlice(signals, func() {
		globalCancel()
		notify.OnceSlice(signals, func() {
			os.Exit(1)
		})
	})
}

func main() {
	restConfig, err := clientcmd.BuildConfigFromFlags(master, kubeconfig)
	if err != nil {
		log.Error(err, "failed to create kubernetes client")
		os.Exit(1)
	}

	control := controller.NewController(&controller.ControllerConfig{
		Logger:    log.WithName("controller"),
		Config:    restConfig,
		Namespace: namespace,
	})

	err = control.Run(ctx)
	if err != nil {
		log.Error(err, "unable to start main controller")
		os.Exit(1)
	}
}
