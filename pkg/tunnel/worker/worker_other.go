//go:build !windows
// +build !windows

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

package worker

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/go-logr/logr"
	"github.com/wzshiming/bridge/chain"
	"github.com/wzshiming/bridge/config"
)

func RunWithReload(ctx context.Context, log logr.Logger, configs []string, dump bool) {
	signalCh := make(chan os.Signal, 2)
	signal.Notify(signalCh, syscall.SIGHUP)
	reloadCn := make(chan struct{}, 1)
	go func() {
		for range signalCh {
			select {
			case reloadCn <- struct{}{}:
			default:
			}
		}
	}()

	wg := sync.WaitGroup{}
	defer wg.Wait()
	var lastWorking = map[string]func(){}
	var cleanups []func()
	count := 1
	reloadCn <- struct{}{}
	for {
		select {
		case <-ctx.Done():
			return
		case <-reloadCn:
		}
		log := log.WithValues("reload_count", count)
		tasks, err := config.LoadConfig(configs...)
		if err != nil {
			for {
				log.Error(err, "LoadConfig")
				log.Info("Try reload again after 1 second")
				time.Sleep(time.Second)
				tasks, err = config.LoadConfig(configs...)
				if err == nil {
					break
				}
			}
		}
		working := map[string]func(){}
		for _, task := range tasks {
			uniq := task.Unique()

			cleanup := lastWorking[uniq]
			if cleanup != nil {
				working[uniq] = cleanup
				continue
			}

			ctx, cancel := context.WithCancel(ctx)
			working[uniq] = cancel
			wg.Add(1)
			go func(ctx context.Context, task config.Chain) {
				defer wg.Done()
				log.Info(chain.ShowChainWithConfig(task))
				for ctx.Err() == nil {
					b := chain.NewBridge(log, dump)
					err := b.BridgeWithConfig(ctx, task)
					if err != nil {
						log.Error(err, "BridgeWithConfig")
					}
					time.Sleep(time.Second)
				}
			}(ctx, task)
		}

		for uniq := range lastWorking {
			if _, ok := working[uniq]; !ok {
				cancel := lastWorking[uniq]
				if cancel != nil {
					cleanups = append(cleanups, cancel)
				}
			}
		}
		lastWorking = working

		// TODO: wait for all task is working
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Second):
		}

		if len(cleanups) > 0 {
			for _, cleanup := range cleanups {
				cleanup()
			}
			cleanups = cleanups[:0]
		}
		count++
	}
}
