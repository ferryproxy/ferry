package second

import (
	_ "embed"

	"github.com/ferry-proxy/ferry/pkg/ferryctl/utils"
)

type BuildJoinConfig struct {
	ControlPlaneTunnelAddress string
	DataPlaneApiserverPort    string
	ControlPlaneIdentity      string
	DataPlaneIdentity         string
	DataPlaneAuthorized       string
	DataPlaneHostkey          string
}

func BuildJoin(conf BuildJoinConfig) (string, error) {
	return utils.RenderString(joinYaml, conf), nil
}

//go:embed join.yaml
var joinYaml string
