package export

import (
	"fmt"
	"github.com/ferry-proxy/ferry/pkg/ferryctl/kubectl"
	"github.com/ferry-proxy/ferry/pkg/ferryctl/log"
	"github.com/ferry-proxy/ferry/pkg/ferryctl/manual"
	"github.com/ferry-proxy/ferry/pkg/ferryctl/vars"
	"github.com/spf13/cobra"
	"net"
)

func NewCommand(logger log.Logger) *cobra.Command {
	var (
		first             = true
		exportHostPort    string
		importServiceName string
		tunnelAddress     = vars.AutoPlaceholders
		reachable         = true
		peerTunnelAddress = vars.AutoPlaceholders
		peerIdentityData  = ""
		bindPort          = vars.AutoPlaceholders
	)
	cmd := &cobra.Command{
		Args: cobra.NoArgs,
		Use:  "export",
		Aliases: []string{
			"e",
		},
		Short: "export commands",
		RunE: func(cmd *cobra.Command, _ []string) (err error) {
			kctl := kubectl.NewKubectl()
			tunnelIdentity := ""
			if !reachable {
				tunnelAddress = ""
			} else {
				if tunnelAddress == vars.AutoPlaceholders {
					tunnelAddress, err = kctl.GetTunnelAddress(cmd.Context())
					if err != nil {
						return err
					}
				}
				tunnelIdentity, err = kctl.GetSecretIdentity(cmd.Context())
				if err != nil {
					return err
				}
			}

			host, port, err := net.SplitHostPort(exportHostPort)
			if err != nil {
				return fmt.Errorf("invalid host and port: %v", err)
			}

			exportTunnelAddress := tunnelAddress
			exportTunnelIdentity := tunnelIdentity
			importTunnelAddress := peerTunnelAddress
			importTunnelIdentity := peerIdentityData
			next := "import"
			isImport := false

			if first {
				return manual.First(cmd.Context(), manual.FirstConfig{
					Next:              next,
					Reachable:         reachable,
					BindPort:          bindPort,
					TunnelAddress:     tunnelAddress,
					TunnelIdentity:    tunnelIdentity,
					ExportPort:        port,
					ExportHost:        host,
					ImportServiceName: importServiceName,
					PeerTunnelAddress: peerTunnelAddress,
				})
			}
			return manual.Second(cmd.Context(), manual.SecondConfig{
				IsImport:             isImport,
				ImportServiceName:    importServiceName,
				BindPort:             bindPort,
				ExportPort:           port,
				ExportHost:           host,
				ExportHubName:        "manual",
				Reachable:            reachable,
				ImportTunnelAddress:  importTunnelAddress,
				ImportTunnelIdentity: importTunnelIdentity,
				ExportTunnelAddress:  exportTunnelAddress,
				ExportTunnelIdentity: exportTunnelIdentity,
			})
		},
	}

	flags := cmd.Flags()
	flags.BoolVar(&first, "first", first, "first step")
	flags.StringVar(&exportHostPort, "export-host-port", "", "host:port")
	flags.StringVar(&importServiceName, "import-service-name", "", "service name")
	flags.StringVar(&tunnelAddress, "tunnel-address", tunnelAddress, "tunnel address")
	flags.BoolVar(&reachable, "reachable", reachable, "whether the tunnel is reachable")
	flags.StringVar(&peerIdentityData, "peer-identity-data", peerIdentityData, "peer identity data")
	flags.StringVar(&peerTunnelAddress, "peer-tunnel-address", peerTunnelAddress, "peer tunnel address")
	flags.StringVar(&bindPort, "bind-port", bindPort, "bind port")
	return cmd
}
