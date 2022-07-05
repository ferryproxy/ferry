package third

import (
	_ "embed"

	"github.com/ferry-proxy/ferry/pkg/ferryctl/utils"
)

type BuildClusterInformationConfig struct {
	DataPlaneReachable             bool
	DataPlaneName                  string
	DataPlaneTunnelAddress         string
	DataPlaneNavigationClusterName string
	DataPlaneReceptionClusterName  string
	DataPlaneKubeconfig            []byte
}

func BuildClusterInformation(conf BuildClusterInformationConfig) (string, error) {
	ci := utils.RenderString(joinInformationYaml, conf)
	return ci, nil
}

//go:embed join_information.yaml
var joinInformationYaml string
