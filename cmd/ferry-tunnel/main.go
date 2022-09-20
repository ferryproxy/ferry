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

	"github.com/ferryproxy/ferry/pkg/tunnel/worker"
	"github.com/ferryproxy/ferry/pkg/utils/signals"
	"github.com/go-logr/zapr"
	flag "github.com/spf13/pflag"
	"github.com/wzshiming/bridge/logger"
	"go.uber.org/zap"
)

var (
	configs []string
	dump    bool
)

func init() {
	flag.StringSliceVarP(&configs, "config", "c", nil, "load from config and ignore --bind and --proxy")
	flag.BoolVarP(&dump, "debug", "d", dump, "Output the communication data.")
	flag.Parse()

	logConfig := zap.NewDevelopmentConfig()
	zapLog, err := logConfig.Build()
	if err != nil {
		logger.Std.Error(err, "who watches the watchmen")
		os.Exit(1)
	}
	logger.Std = zapr.NewLogger(zapLog).WithName("ferry-tunnel")
}

func main() {
	stopCh := signals.SetupNotifySignalHandler()
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		<-stopCh
		cancel()
	}()

	worker.RunWithReload(ctx, logger.Std, configs, dump)
	return
}
