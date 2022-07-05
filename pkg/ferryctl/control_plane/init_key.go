package control_plane

import (
	_ "embed"

	"github.com/ferry-proxy/ferry/pkg/ferryctl/utils"
)

type BuildInitKeyConfig struct {
	ClusterName string
	Identity    string
	Authorized  string
	Hostkey     string
}

func BuildInitKey(conf BuildInitKeyConfig) (string, error) {
	return utils.RenderString(initKeyYaml, conf), nil
}

//go:embed init_key.yaml
var initKeyYaml string
