package manual

import (
	"context"
	"encoding/base64"
	"fmt"
	"net"
	"strings"

	"github.com/ferry-proxy/ferry/pkg/ferryctl/kubectl"
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

func Second(ctx context.Context, config SecondConfig) error {
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
			return fmt.Errorf("invalid service and port: %v", err)
		}
		conf.ImportTunnelHost = importTunnelHost
		conf.ImportTunnelPort = importTunnelPort
		conf.ImportTunnelIdentity = config.ImportTunnelIdentity
	} else {
		exportTunnelHost, exportTunnelPort, err := net.SplitHostPort(config.ExportTunnelAddress)
		if err != nil {
			return fmt.Errorf("invalid service and port: %v", err)
		}
		conf.ExportTunnelHost = exportTunnelHost
		conf.ExportTunnelPort = exportTunnelPort
		conf.ExportTunnelIdentity = config.ExportTunnelIdentity
	}

	exportPortResource, importPortResource, importAddress, err := BuildManualPort(conf)
	if err != nil {
		return fmt.Errorf("failed to build manual port: %v", err)
	}

	applyResource := ""
	otherResource := ""

	if config.IsImport {
		applyResource = importPortResource
		otherResource = exportPortResource
	} else {
		applyResource = exportPortResource
		otherResource = importPortResource
	}
	if applyResource != "" {
		kctl := kubectl.NewKubectl()
		err = kctl.ApplyWithReader(ctx, strings.NewReader(applyResource))
		if err != nil {
			return fmt.Errorf("failed to apply reousrce: %v", err)
		}
	}

	if otherResource != "" {
		baseCmd := base64.StdEncoding.EncodeToString([]byte(otherResource))
		fmt.Printf("# ++++ Please run the following command to peer tunnel:\n")
		fmt.Printf("# =============================================\n")
		fmt.Printf("echo %s | base64 --decode | kubectl apply -f -\n", baseCmd)
		fmt.Printf("# =============================================\n")
		fmt.Printf("# The service will be available after executing the above on the peer tunnel:\n")
		fmt.Printf("# Service: %s\n", importAddress)
	} else {
		fmt.Printf("# This service is already available:\n")
		fmt.Printf("# Service: %s\n", importAddress)
	}
	return nil
}
