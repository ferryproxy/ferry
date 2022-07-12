package data_plane

import (
	_ "embed"

	"github.com/ferryproxy/ferry/pkg/ferryctl/utils"
)

type BuildInitKeyConfig struct {
	Identity   string
	Authorized string
	Hostkey    string
}

func BuildInitKey(conf BuildInitKeyConfig) (string, error) {
	return utils.RenderString(keyYaml, conf), nil
}

//go:embed init_key.yaml
var keyYaml string
