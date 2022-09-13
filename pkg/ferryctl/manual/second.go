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
	"encoding/base64"
	"fmt"
	"strconv"
	"time"

	"github.com/ferryproxy/api/apis/traffic/v1alpha2"
	"github.com/ferryproxy/ferry/pkg/ferry-controller/router/resource"
)

type SecondConfig struct {
	RouteName              string
	ImportHub              string
	ExportHub              string
	IsImport               bool
	ImportService          string
	BindPort               string
	ExportPort             string
	ExportService          string
	Reachable              bool
	ImportTunnelAddress    string
	ImportTunnelAuthorized string
	ExportTunnelAddress    string
	ExportTunnelAuthorized string
}

func Second(conf SecondConfig) (applyResource, otherResource, importAddress string, err error) {
	bindPort, err := strconv.Atoi(conf.BindPort)
	if err != nil {
		return "", "", "", err
	}
	port, err := strconv.Atoi(conf.ExportPort)
	if err != nil {
		return "", "", "", err
	}

	suffix := time.Now().Format("20060102150405")
	exportHubName := conf.ExportHub
	if exportHubName == "" {
		exportHubName = fmt.Sprintf("manual-export-%s", suffix)
	}
	importHubName := conf.ImportHub
	if importHubName == "" {
		importHubName = fmt.Sprintf("manual-import-%s", suffix)
	}
	routeName := conf.RouteName
	if routeName == "" {
		routeName = fmt.Sprintf("manual-%s", suffix)
	}
	importAuthorized, err := base64.StdEncoding.DecodeString(conf.ImportTunnelAuthorized)
	if err != nil {
		return "", "", "", err
	}

	exportAuthorized, err := base64.StdEncoding.DecodeString(conf.ExportTunnelAuthorized)
	if err != nil {
		return "", "", "", err
	}

	importName, importNamespace := GetService(conf.ImportService)
	exportName, exportNamespace := GetService(conf.ExportService)
	mc := ManualConfig{
		RouteName:       routeName,
		ImportHubName:   importHubName,
		ImportName:      importName,
		ImportNamespace: importNamespace,
		ImportGateway: v1alpha2.HubSpecGateway{
			Reachable: conf.ImportTunnelAddress != "",
			Address:   conf.ImportTunnelAddress,
		},
		ImportAuthorized: string(importAuthorized),
		BindPort:         int32(bindPort),
		Port:             int32(port),
		ExportHubName:    exportHubName,
		ExportName:       exportName,
		ExportNamespace:  exportNamespace,
		ExportGateway: v1alpha2.HubSpecGateway{
			Reachable: conf.ExportTunnelAddress != "",
			Address:   conf.ExportTunnelAddress,
		},
		ExportAuthorized: string(exportAuthorized),
	}
	m := NewManual(mc)
	resources, err := m.BuildResource()
	if err != nil {
		return "", "", "", err
	}

	if len(resources) == 0 {
		return "", "", "", fmt.Errorf("failed build resource: output is empty")
	}

	importResource, err := resource.MarshalYAML(resources[importHubName]...)
	if err != nil {
		return "", "", "", err
	}
	exportResource, err := resource.MarshalYAML(resources[exportHubName]...)
	if err != nil {
		return "", "", "", err
	}

	if conf.IsImport {
		applyResource = string(importResource)
		otherResource = string(exportResource)
	} else {
		applyResource = string(exportResource)
		otherResource = string(importResource)
	}

	importAddress = fmt.Sprintf("%s.svc:%s", conf.ImportService, conf.ExportPort)

	return applyResource, otherResource, importAddress, nil
}
