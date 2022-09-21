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
	"time"

	"github.com/ferryproxy/ferry/pkg/services/registry/client"
	"github.com/ferryproxy/ferry/pkg/utils/env"
	"github.com/ferryproxy/ferry/pkg/utils/signals"
	"github.com/go-logr/zapr"
	"go.uber.org/zap"
)

var (
	registerServiceURL = env.GetEnv("REGISTER_SERVICE_URL", "")
	hubName            = env.GetEnv("HUB_NAME", "")
)

func main() {
	logConfig := zap.NewDevelopmentConfig()
	zapLog, err := logConfig.Build()
	if err != nil {
		os.Exit(1)
	}
	log := zapr.NewLogger(zapLog).WithName("ferry-joiner")

	stopCh := signals.SetupNotifySignalHandler()
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		<-stopCh
		cancel()
	}()

	if hubName == "" {
		hubName = "joiner-" + time.Now().Format("20060102150405")
	}

	cli := client.NewClient(registerServiceURL)
	err = cli.Create(ctx, hubName)
	if err != nil {
		log.Error(err, "failed to join")
		os.Exit(1)
	}
}
