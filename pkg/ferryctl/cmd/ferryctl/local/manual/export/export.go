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

package export

import (
	"encoding/base64"
	"fmt"
	"net"
	"strings"

	"github.com/ferryproxy/ferry/pkg/ferryctl/kubectl"
	"github.com/ferryproxy/ferry/pkg/ferryctl/log"
	"github.com/ferryproxy/ferry/pkg/ferryctl/manual"
	"github.com/ferryproxy/ferry/pkg/ferryctl/utils"
	"github.com/ferryproxy/ferry/pkg/ferryctl/vars"
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
