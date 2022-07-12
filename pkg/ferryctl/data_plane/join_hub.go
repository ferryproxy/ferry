package data_plane

import (
	_ "embed"

	"github.com/ferryproxy/ferry/pkg/ferryctl/utils"
)

type BuildHubConfig struct {
	DataPlaneReachable     bool
	DataPlaneName          string
	DataPlaneTunnelAddress string
	DataPlaneNavigation    []string
	DataPlaneReception     []string
	DataPlaneKubeconfig    string
}

func BuildHub(conf BuildHubConfig) (string, error) {
	ci := utils.RenderString(joinHubYaml, conf)
	return ci, nil
}

//go:embed join_hub.yaml
var joinHubYaml string
