package manual

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

type FirstConfig struct {
	Next              string
	Reachable         bool
	BindPort          string
	TunnelAddress     string
	TunnelIdentity    string
	ExportPort        string
	ExportHost        string
	ImportServiceName string
	PeerTunnelAddress string
}

func First(ctx context.Context, conf FirstConfig) error {
	tunnelAddress := conf.TunnelAddress
	tunnelIdentity := conf.TunnelIdentity
	exportHost := conf.ExportHost
	exportPort := conf.ExportPort
	importServiceName := conf.ImportServiceName
	bindPort := conf.BindPort
	peerTunnelAddress := conf.PeerTunnelAddress

	args := []string{
		"--first=false",
		"--reachable=" + strconv.FormatBool(!conf.Reachable),
	}
	if conf.Reachable {
		args = append(args, "--peer-identity-data="+tunnelIdentity)
	} else {
		tunnelAddress = ""
	}
	args = append(args, "--peer-tunnel-address="+tunnelAddress)

	args = append(args, "--export-host-port="+exportHost+":"+exportPort)

	if importServiceName == "" {
		importServiceName = fmt.Sprintf("%s-%s", strings.ReplaceAll(exportHost, ".", "-"), exportPort)
	}
	args = append(args, "--bind-port="+bindPort)
	args = append(args, "--import-service-name="+importServiceName)
	args = append(args, "--tunnel-address="+peerTunnelAddress)

	fmt.Printf("# ++++ Please run the following command to peer tunnel:\n")
	fmt.Printf("# =============================================\n")
	fmt.Printf("ferryctl data-plane init\n")
	fmt.Printf("ferryctl local manual %s %s\n",
		conf.Next,
		strings.Join(args, " "),
	)
	fmt.Printf("# =============================================\n")
	return nil
}
