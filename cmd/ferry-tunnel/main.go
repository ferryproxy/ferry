package main

import (
	"context"
	"os"
	"syscall"

	"github.com/ferry-proxy/ferry/pkg/consts"
	"github.com/ferry-proxy/ferry/pkg/ferry-tunnel/controller"
	"github.com/ferry-proxy/ferry/pkg/utils/env"
	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"github.com/wzshiming/notify"
	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	ctx, globalCancel = context.WithCancel(context.Background())
	log               logr.Logger
	serviceName       = env.GetEnv("SERVICE_NAME", consts.FerryTunnelName)
	namespace         = env.GetEnv("NAMESPACE", consts.FerryTunnelNamespace)
	labelSelector     = env.GetEnv("LABEL_SELECTOR", "tunnel.ferry.zsm.io/service=inject")
	master            = env.GetEnv("MASTER", "")
	kubeconfig        = env.GetEnv("KUBECONFIG", "")
	conf              = "./bridge.conf"
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
