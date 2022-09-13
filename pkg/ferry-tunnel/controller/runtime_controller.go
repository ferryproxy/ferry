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

package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/ferryproxy/ferry/pkg/consts"
	"github.com/ferryproxy/ferry/pkg/utils/trybuffer"
	"github.com/go-logr/logr"
	"k8s.io/client-go/kubernetes"
)

type RuntimeController struct {
	cmd           *exec.Cmd
	chains        []json.RawMessage
	try           *trybuffer.TryBuffer
	mut           sync.Mutex
	namespace     string
	labelSelector string
	logger        logr.Logger
	clientset     kubernetes.Interface
}

type RuntimeControllerConfig struct {
	Namespace     string
	LabelSelector string
	Logger        logr.Logger
	Clientset     kubernetes.Interface
}

func NewRuntimeController(conf *RuntimeControllerConfig) *RuntimeController {
	return &RuntimeController{
		namespace:     conf.Namespace,
		labelSelector: conf.LabelSelector,
		logger:        conf.Logger,
		clientset:     conf.Clientset,
	}
}

func (r *RuntimeController) Run(ctx context.Context) error {
	_, err := os.Stat(consts.TunnelRulesConfigPath)
	if err != nil {
		err = atomicWrite(consts.TunnelRulesConfigPath, []byte(`{}`), 0644)
		if err != nil {
			r.logger.Error(err, "failed to write config file")
			return err
		}
	}

	r.try = trybuffer.NewTryBuffer(func() {
		err := r.reload()
		if err != nil {
			r.logger.Error(err, "reload")
		}
	}, time.Second/10)

	go r.watch(ctx)

	go func() {
		<-ctx.Done()
		r.cmd.Process.Signal(syscall.SIGTERM)
	}()

	r.logger.Info("Start ferry tunnel")
	for ctx.Err() == nil {
		err := r.runtime(ctx)
		if err != nil {
			r.logger.Error(err, "tunnel exited")
		}
		time.Sleep(5 * time.Second)
	}

	r.try.Close()
	return nil
}

func (r *RuntimeController) watch(ctx context.Context) {
	backoff := time.Second
	r.logger.Info("starting watcher", "namespace", r.namespace, "labelSelector", r.labelSelector)
	for ctx.Err() == nil {
		watcher := NewConfigWatcher(&ConfigWatcherConfig{
			Clientset:     r.clientset,
			Logger:        r.logger.WithName("config-watch"),
			Namespace:     r.namespace,
			LabelSelector: r.labelSelector,
			ReloadFunc: func(chains []json.RawMessage) {
				backoff = time.Second
				r.mut.Lock()
				defer r.mut.Unlock()
				r.chains = chains
				r.try.Try()
			},
		})
		err := watcher.Run(ctx)
		if err != nil {
			r.logger.Error(err, "failed to watch")
		}
		time.Sleep(backoff)
		backoff <<= 1
		if backoff > time.Minute {
			backoff = time.Minute
		}
	}
}

func (r *RuntimeController) runtime(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "ferry-tunnel", "-c", consts.TunnelRulesConfigPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	r.mut.Lock()
	r.cmd = cmd
	r.mut.Unlock()
	return cmd.Run()
}

func (r *RuntimeController) reload() error {
	r.mut.Lock()
	defer r.mut.Unlock()

	tunnelConfig, err := json.Marshal(struct {
		Chains []json.RawMessage `json:"chains"`
	}{
		Chains: r.chains,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal tunnel config: %w", err)
	}
	r.logger.V(1).Info("Reload", "config", r)

	err = atomicWrite(consts.TunnelRulesConfigPath, tunnelConfig, 0644)
	if err != nil {
		return fmt.Errorf("failed to write tunnel config: %w", err)
	}

	if r.cmd != nil {
		err = r.cmd.Process.Signal(syscall.SIGHUP)
		if err != nil {
			return fmt.Errorf("failed to emit signal to tunnel: %w", err)
		}
	}
	return nil
}

func atomicWrite(path string, data []byte, mode os.FileMode) error {
	f, err := ioutil.TempFile(filepath.Dir(path), ".tmp-"+filepath.Base(path))
	if err != nil {
		return fmt.Errorf("create tmp file : %v", err)
	}
	err = os.Chmod(f.Name(), mode)
	if err != nil {
		f.Close()
		return fmt.Errorf("changes the mode of the tmp file : %v", err)
	}
	_, err = f.Write(data)
	f.Close()
	if err != nil {
		return fmt.Errorf("write atomic data: %v", err)
	}
	return os.Rename(f.Name(), path)
}
