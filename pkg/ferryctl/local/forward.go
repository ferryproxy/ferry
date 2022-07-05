package local

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/ferry-proxy/ferry/pkg/ferryctl/bridge"
	"github.com/ferry-proxy/ferry/pkg/ferryctl/kubectl"
)

func ForwardDial(ctx context.Context, local string, remote string) error {
	b := bridge.NewBridge()
	return b.ForwardDial(ctx, local, remote)
}

func ForwardListen(ctx context.Context, remote string, local string) error {
	remoteAddress, remotePort, err := net.SplitHostPort(remote)
	service := strings.Split(remoteAddress, ".")
	if len(service) < 2 {
		return fmt.Errorf("invalid remote address %s", remote)
	}

	kctl := kubectl.NewKubectl()
	port, err := kctl.GetUnusedPort(ctx)
	if err != nil {
		return err
	}

	joinService, err := BuildForwardTCP(BuildForwardTCPConfig{
		ServiceName:      service[0],
		ServiceNamespace: service[1],
		Port:             remotePort,
		TargetPort:       port,
	})
	if err != nil {
		return err
	}

	err = kctl.ApplyWithReader(ctx, strings.NewReader(joinService))
	if err != nil {
		return err
	}

	b := bridge.NewBridge()
	return b.ForwardListen(ctx, "0.0.0.0:"+port, local)
}
