package manual

import (
	_ "embed"

	"github.com/ferry-proxy/ferry/pkg/ferryctl/utils"
)

type buildExportConfig struct {
	ExportName           string
	ExportNamespace      string
	BindPort             string
	ExportPort           string
	ExportHost           string
	ImportTunnelHost     string
	ImportTunnelPort     string
	ImportTunnelIdentity string
}

func buildExport(conf buildExportConfig) (string, error) {
	return utils.RenderString(exportYaml, conf), nil
}

//go:embed export.yaml
var exportYaml string
