package first

import (
	_ "embed"

	"github.com/ferry-proxy/ferry/pkg/ferryctl/utils"
)

type BuildPreJoinTunnelConfig struct {
	DataPlaneName          string
	DataPlaneApiserverPort string
	DataPlaneTunnelAddress string
	DataPlaneIdentity      string
}

func BuildPreJoinTunnel(conf BuildPreJoinTunnelConfig) (string, error) {
	return utils.RenderString(preJoinTunnelYaml, conf), nil
}

//go:embed pre_join_tunnel.yaml
var preJoinTunnelYaml string
