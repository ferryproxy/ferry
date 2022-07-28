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
	_ "embed"

	"github.com/ferryproxy/ferry/pkg/ferryctl/utils"
)

type BuildInitTunnelConfig struct {
	Image             string
	TunnelServiceType string // LoadBalancer or NodePort
}

func BuildInitTunnel(conf BuildInitTunnelConfig) (string, error) {
	return utils.RenderString(tunnelYaml, conf), nil
}

//go:embed init_tunnel.yaml
var tunnelYaml string
