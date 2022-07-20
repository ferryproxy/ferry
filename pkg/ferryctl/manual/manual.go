/*
Copyright 2022 FerryProxy Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package manual

import (
	"fmt"
	"strings"

	"github.com/ferryproxy/ferry/pkg/consts"
)

type BuildManualPortConfig struct {
	ImportServiceName string

	BindPort   string
	ExportPort string
	ExportHost string

	ExportHubName string

	ExportTunnelHost     string
	ExportTunnelPort     string
	ExportTunnelIdentity string

	ImportTunnelHost     string
	ImportTunnelPort     string
	ImportTunnelIdentity string
}

func BuildManualPort(conf BuildManualPortConfig) (exportPortResource, importPortResource, importAddress string, err error) {
	namespace := consts.FerryTunnelNamespace
	exportName := fmt.Sprintf("%s-%s", conf.ImportServiceName, "export")
	importName := fmt.Sprintf("%s-%s", conf.ImportServiceName, "import")
	exportPortResource, err = buildExport(buildExportConfig{
		ExportName:      exportName,
		ExportNamespace: namespace,
		BindPort:        conf.BindPort,
		ExportPort:      conf.ExportPort,
		ExportHost:      conf.ExportHost,

		ImportTunnelHost:     conf.ImportTunnelHost,
		ImportTunnelPort:     conf.ImportTunnelPort,
		ImportTunnelIdentity: conf.ImportTunnelIdentity,
	})
	if err != nil {
		return "", "", "", err
	}
	importPortResource, err = buildImport(buildImportConfig{
		ImportServiceName: conf.ImportServiceName,
		ImportName:        importName,
		ImportNamespace:   namespace,
		BindPort:          conf.BindPort,
		ExportPort:        conf.ExportPort,
		ExportHost:        conf.ExportHost,

		ExportHubName: conf.ExportHubName,

		ExportTunnelHost:     conf.ExportTunnelHost,
		ExportTunnelPort:     conf.ExportTunnelPort,
		ExportTunnelIdentity: conf.ExportTunnelIdentity,
	})
	if err != nil {
		return "", "", "", err
	}

	exportPortResource = strings.TrimSpace(exportPortResource)
	importPortResource = strings.TrimSpace(importPortResource)

	importAddress = fmt.Sprintf("%s.%s.svc:%s", conf.ImportServiceName, namespace, conf.ExportPort)

	return exportPortResource, importPortResource, importAddress, nil
}
