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

	"github.com/ferry-proxy/ferry/pkg/utils/trybuffer"
	"github.com/go-logr/logr"
	"github.com/wzshiming/notify"
	"k8s.io/client-go/kubernetes"
)

type RuntimeController struct {
	cmd           *exec.Cmd
	chains        []json.RawMessage
	try           *trybuffer.TryBuffer
	mut           sync.Mutex
	conf          string
	namespace     string
	labelSelector string
	logger        logr.Logger
	clientset     kubernetes.Interface
}

type RuntimeControllerConfig struct {
	Conf          string
	Namespace     string
	LabelSelector string
	Logger        logr.Logger
	Clientset     kubernetes.Interface
}

func NewRuntimeController(conf *RuntimeControllerConfig) *RuntimeController {
	return &RuntimeController{
		conf:          conf.Conf,
		namespace:     conf.Namespace,
		labelSelector: conf.LabelSelector,
		logger:        conf.Logger,
		clientset:     conf.Clientset,
	}
}

func (r *RuntimeController) Run(ctx context.Context) error {
	_, err := os.Stat(r.conf)
	if err != nil {
		err = atomicWrite(r.conf, []byte(`{}`), 0644)
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
	}, time.Second/2)

	go r.watch(ctx)

	signals := []os.Signal{syscall.SIGINT, syscall.SIGTERM}
	notify.OnceSlice(signals, func() {
		r.mut.Lock()
		defer r.mut.Unlock()
		r.cmd.Process.Signal(syscall.SIGTERM)
	})

	r.logger.Info("Start ferry tunnel")
	for ctx.Err() == nil {
		err := r.runtime(ctx)
		if err != nil {
			r.logger.Error(err, "bridge exited")
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
			ReloadFunc: func(d []json.RawMessage) {
				backoff = time.Second
				r.mut.Lock()
				defer r.mut.Unlock()
				r.chains = d
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
	cmd := exec.CommandContext(ctx, "bridge", "-c", r.conf)
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

	bridgeConfig, err := json.MarshalIndent(struct {
		Chains []json.RawMessage `json:"chains"`
	}{
		Chains: r.chains,
	}, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal bridge config: %w", err)
	}
	r.logger.V(1).Info("Reload", "config", r)

	err = atomicWrite(r.conf, bridgeConfig, 0644)
	if err != nil {
		return fmt.Errorf("failed to write bridge config: %w", err)
	}

	if r.cmd != nil {
		err = r.cmd.Process.Signal(syscall.SIGHUP)
		if err != nil {
			return fmt.Errorf("failed to emit signal to bridge: %w", err)
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
