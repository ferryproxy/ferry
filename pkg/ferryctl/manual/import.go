package manual

import (
	_ "embed"

	"github.com/ferryproxy/ferry/pkg/ferryctl/utils"
)

type buildImportConfig struct {
	ImportServiceName string
	ImportName        string
	ImportNamespace   string
	BindPort          string
	ExportPort        string
	ExportHost        string
	ExportHubName     string

	ExportTunnelHost     string
	ExportTunnelPort     string
	ExportTunnelIdentity string
}

func buildImport(conf buildImportConfig) (string, error) {
	return utils.RenderString(importYaml, conf), nil
}

//go:embed import.yaml
var importYaml string
