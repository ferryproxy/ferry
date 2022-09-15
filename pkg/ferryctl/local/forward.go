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

package local

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/ferryproxy/ferry/pkg/ferryctl/bridge"
	"github.com/ferryproxy/ferry/pkg/ferryctl/kubectl"
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
	return b.ForwardListen(ctx, ":"+port, local)
}
