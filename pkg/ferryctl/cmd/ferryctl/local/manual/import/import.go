package import_cmd

import (
	"encoding/base64"
	"fmt"
	"net"
	"strings"

	"github.com/ferry-proxy/ferry/pkg/ferryctl/kubectl"
	"github.com/ferry-proxy/ferry/pkg/ferryctl/log"
	"github.com/ferry-proxy/ferry/pkg/ferryctl/manual"
	"github.com/ferry-proxy/ferry/pkg/ferryctl/utils"
	"github.com/ferry-proxy/ferry/pkg/ferryctl/vars"
	"github.com/spf13/cobra"
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
		Use:  "import",
		Aliases: []string{
			"i",
		},
		Short: "import commands",
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

			if bindPort == vars.AutoPlaceholders {
				bindPort, err = kctl.GetUnusedPort(cmd.Context())
				if err != nil {
					return err
				}
			}

			exportTunnelAddress := peerTunnelAddress
			exportTunnelIdentity := peerIdentityData
			importTunnelAddress := tunnelAddress
			importTunnelIdentity := tunnelIdentity
			next := "export"
			isImport := true

			if first {
				second, err := manual.First(cmd.Context(), manual.FirstConfig{
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

				if err != nil {
					return err
				}

				utils.Prompt(
					"peer tunnel",
					"ferryctl data-plane init",
					second,
				)
				return nil
			}
			applyResource, otherResource, importAddress, err := manual.Second(cmd.Context(), manual.SecondConfig{
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

			if applyResource != "" {
				err = kctl.ApplyWithReader(cmd.Context(), strings.NewReader(applyResource))
				if err != nil {
					return fmt.Errorf("failed to apply reousrce: %v", err)
				}
			}

			if otherResource != "" {
				baseCmd := base64.StdEncoding.EncodeToString([]byte(otherResource))
				utils.Prompt(
					"peer tunnel",
					fmt.Sprintf("echo %s | base64 --decode | kubectl apply -f -\n", baseCmd),
				)
				fmt.Printf("# The service will be available after executing the above on the peer tunnel:\n")
				fmt.Printf("# Service: %s\n", importAddress)
			} else {
				fmt.Printf("# This service is already available:\n")
				fmt.Printf("# Service: %s\n", importAddress)
			}
			return nil
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
