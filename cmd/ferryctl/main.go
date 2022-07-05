package main

import (
	"context"
	"log"
	"os"
	"syscall"

	"github.com/ferry-proxy/ferry/pkg/ferryctl/cmd/ferryctl"
	"github.com/wzshiming/notify"
)

var (
	ctx, globalCancel = context.WithCancel(context.Background())
)

func init() {
	signals := []os.Signal{syscall.SIGINT, syscall.SIGTERM}
	notify.OnceSlice(signals, func() {
		globalCancel()
		notify.OnceSlice(signals, func() {
			os.Exit(1)
		})
	})
}

func main() {
	logger := log.New(os.Stdout, "", 0)
	cmd := ferryctl.NewCommand(logger)
	err := cmd.ExecuteContext(ctx)
	if err != nil {
		logger.Println(err)
		os.Exit(1)
	}
}
