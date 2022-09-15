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

package data_plane

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/ferryproxy/ferry/pkg/ferryctl/kubectl"
	"github.com/ferryproxy/ferry/pkg/ferryctl/utils"
)

type ClusterInitConfig struct {
	FerryTunnelImage  string
	TunnelServiceType string // LoadBalancer or NodePort
}

func ClusterInit(ctx context.Context, conf ClusterInitConfig) error {
	kctl := kubectl.NewKubectl()
	fmt.Fprintf(os.Stderr, "ferry tunnel image: %s\n", conf.FerryTunnelImage)
	tunnel, err := BuildInitTunnel(BuildInitTunnelConfig{
		Image:             conf.FerryTunnelImage,
		TunnelServiceType: conf.TunnelServiceType,
	})
	if err != nil {
		return err
	}
	err = kctl.ApplyWithReader(ctx, strings.NewReader(tunnel))
	if err != nil {
		return err
	}

	var authorized string
	identity, _ := kctl.GetSecretIdentity(ctx)
	if identity != "" {
		authorized, _ = kctl.GetSecretAuthorized(ctx)
	}

	if identity == "" || authorized == "" {
		identity, authorized, err = utils.GetKey()
		if err != nil {
			return err
		}
	}

	key, err := BuildInitKey(BuildInitKeyConfig{
		Identity:   identity,
		Authorized: authorized,
	})
	if err != nil {
		return err
	}
	err = kctl.ApplyWithReader(ctx, strings.NewReader(key))
	if err != nil {
		return err
	}

	err = kctl.ApplyWithReader(ctx, strings.NewReader(joinRBACYaml))
	if err != nil {
		return err
	}
	return nil
}
