package third

import (
	_ "embed"

	"github.com/ferry-proxy/ferry/pkg/ferryctl/utils"
)

type BuildHubConfig struct {
	DataPlaneReachable         bool
	DataPlaneName              string
	DataPlaneTunnelAddress     string
	DataPlaneNavigationHubName string
	DataPlaneReceptionHubName  string
	DataPlaneKubeconfig        []byte
}

func BuildHub(conf BuildHubConfig) (string, error) {
	ci := utils.RenderString(joinInformationYaml, conf)
	return ci, nil
}

//go:embed join_information.yaml
var joinInformationYaml string
