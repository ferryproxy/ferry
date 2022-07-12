package data_plane

import (
	_ "embed"

	"github.com/ferryproxy/ferry/pkg/ferryctl/utils"
)

type BuildInitTunnelConfig struct {
	Image string
}

func BuildInitTunnel(conf BuildInitTunnelConfig) (string, error) {
	return utils.RenderString(tunnelYaml, conf), nil
}

//go:embed init_tunnel.yaml
var tunnelYaml string
