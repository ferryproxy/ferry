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
	"context"
	"fmt"
	"net"
)

type SecondConfig struct {
	IsImport             bool
	ImportServiceName    string
	BindPort             string
	ExportPort           string
	ExportHost           string
	ExportHubName        string
	Reachable            bool
	ImportTunnelAddress  string
	ImportTunnelIdentity string
	ExportTunnelAddress  string
	ExportTunnelIdentity string
}

func Second(ctx context.Context, config SecondConfig) (applyResource, otherResource, importAddress string, err error) {
	conf := BuildManualPortConfig{
		ImportServiceName: config.ImportServiceName,
		BindPort:          config.BindPort,
		ExportPort:        config.ExportPort,
		ExportHost:        config.ExportHost,
		ExportHubName:     config.ExportHubName,
	}

	if config.Reachable == config.IsImport {
		importTunnelHost, importTunnelPort, err := net.SplitHostPort(config.ImportTunnelAddress)
		if err != nil {
			return "", "", "", fmt.Errorf("invalid service and port: %v", err)
		}
		conf.ImportTunnelHost = importTunnelHost
		conf.ImportTunnelPort = importTunnelPort
		conf.ImportTunnelIdentity = config.ImportTunnelIdentity
	} else {
		exportTunnelHost, exportTunnelPort, err := net.SplitHostPort(config.ExportTunnelAddress)
		if err != nil {
			return "", "", "", fmt.Errorf("invalid service and port: %v", err)
		}
		conf.ExportTunnelHost = exportTunnelHost
		conf.ExportTunnelPort = exportTunnelPort
		conf.ExportTunnelIdentity = config.ExportTunnelIdentity
	}

	exportPortResource, importPortResource, importAddress, err := BuildManualPort(conf)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to build manual port: %v", err)
	}

	if config.IsImport {
		applyResource = importPortResource
		otherResource = exportPortResource
	} else {
		applyResource = exportPortResource
		otherResource = importPortResource
	}

	return applyResource, otherResource, importAddress, nil
}
