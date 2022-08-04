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
	"sync"

	_ "github.com/wzshiming/bridge/protocols/command"
	_ "github.com/wzshiming/bridge/protocols/connect"
	_ "github.com/wzshiming/bridge/protocols/netcat"
	_ "github.com/wzshiming/bridge/protocols/socks4"
	_ "github.com/wzshiming/bridge/protocols/socks5"
	_ "github.com/wzshiming/bridge/protocols/ssh"
	_ "github.com/wzshiming/bridge/protocols/tls"

	_ "github.com/wzshiming/anyproxy/pprof"
	_ "github.com/wzshiming/anyproxy/proxies/httpproxy"
	_ "github.com/wzshiming/anyproxy/proxies/shadowsocks"
	_ "github.com/wzshiming/anyproxy/proxies/socks4"
	_ "github.com/wzshiming/anyproxy/proxies/socks5"
	_ "github.com/wzshiming/anyproxy/proxies/sshproxy"

	"github.com/ferryproxy/ferry/pkg/utils/signals"
	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	flag "github.com/spf13/pflag"
	"github.com/wzshiming/bridge/chain"
	"github.com/wzshiming/bridge/config"
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

	runWithReload(ctx, logger.Std, configs)
	return
}

func run(ctx context.Context, log logr.Logger, tasks []config.Chain) {
	var wg sync.WaitGroup
	wg.Add(len(tasks))
	for _, task := range tasks {
		go func(task config.Chain) {
			defer wg.Done()
			log.Info(chain.ShowChainWithConfig(task))
			b := chain.NewBridge(log, dump)
			err := b.BridgeWithConfig(ctx, task)
			if err != nil {
				log.Error(err, "BridgeWithConfig")
			}
		}(task)
	}
	wg.Wait()
}
