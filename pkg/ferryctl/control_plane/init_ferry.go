package control_plane

import (
	_ "embed"

	"github.com/ferryproxy/ferry/pkg/ferryctl/utils"
)

type BuildInitFerryConfig struct {
	Image string
}

func BuildInitFerry(conf BuildInitFerryConfig) (string, error) {
	return utils.RenderString(ferryYaml, conf), nil
}

//go:embed init_ferry.yaml
var ferryYaml string
