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
