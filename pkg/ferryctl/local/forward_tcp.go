package local

import (
	_ "embed"

	"github.com/ferryproxy/ferry/pkg/ferryctl/utils"
)

type BuildForwardTCPConfig struct {
	ServiceName      string
	ServiceNamespace string
	Port             string
	TargetPort       string
}

func BuildForwardTCP(conf BuildForwardTCPConfig) (string, error) {
	return utils.RenderString(forwardTcpYaml, conf), nil
}

//go:embed forward_tcp.yaml
var forwardTcpYaml string
