package kubectl

import (
	_ "embed"

	"github.com/ferryproxy/ferry/pkg/ferryctl/utils"
)

type BuildKubeconfigConfig struct {
	Name             string
	ApiserverAddress string
	Token            string
}

func BuildKubeconfig(conf BuildKubeconfigConfig) (string, error) {
	return utils.RenderString(kubeconfigYaml, conf), nil
}

//go:embed kubeconfig.yaml
var kubeconfigYaml string
