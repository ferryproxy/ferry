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
		routeName          string
		importHub          string
		exportHub          string
		first              = true
		exportService      string
		importService      string
		tunnelAddress      = vars.AutoPlaceholders
		reachable          = true
		peerTunnelAddress  = vars.AutoPlaceholders
		peerAuthorizedData = ""
		bindPort           = vars.AutoPlaceholders
		port               string
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
			tunnelAuthorized := ""
			if !reachable {
				tunnelAddress = ""
			} else {
				if tunnelAddress == vars.AutoPlaceholders {
					tunnelAddress, err = kctl.GetTunnelAddress(cmd.Context())
					if err != nil {
						return err
					}
				}
			}

			tunnelAuthorized, err = kctl.GetSecretAuthorized(cmd.Context())
			if err != nil {
				return err
			}
			if tunnelAuthorized == "" {
				return fmt.Errorf("failed get authorized key")
			}

			exportService = manual.FormatService(exportService)
			importService = manual.FormatService(importService)
			exportTunnelAddress := tunnelAddress
			exportTunnelAuthorized := tunnelAuthorized
			importTunnelAddress := peerTunnelAddress
			importTunnelAuthorized := peerAuthorizedData
			next := "import"
			isImport := false

			if first {
				second, err := manual.First(manual.FirstConfig{
					RouteName:         routeName,
					ExportHub:         exportHub,
					ImportHub:         importHub,
					Next:              next,
					Reachable:         reachable,
					BindPort:          bindPort,
					TunnelAddress:     tunnelAddress,
					TunnelAuthorized:  tunnelAuthorized,
					ExportPort:        port,
					ExportService:     exportService,
					ImportService:     importService,
					PeerTunnelAddress: peerTunnelAddress,
				})
				if err != nil {
					return err
				}

				utils.Prompt(
					"peer tunnel",
					second,
				)
				return nil
			}
			applyResource, otherResource, importAddress, err := manual.Second(manual.SecondConfig{
				RouteName:              routeName,
				ExportHub:              exportHub,
				ImportHub:              importHub,
				IsImport:               isImport,
				ImportService:          importService,
				BindPort:               bindPort,
				ExportPort:             port,
				ExportService:          exportService,
				Reachable:              reachable,
				ImportTunnelAddress:    importTunnelAddress,
				ImportTunnelAuthorized: importTunnelAuthorized,
				ExportTunnelAddress:    exportTunnelAddress,
				ExportTunnelAuthorized: exportTunnelAuthorized,
			})
			if err != nil {
				return err
			}

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
	flags.StringVar(&routeName, "route-name", "", "route name")
	flags.StringVar(&exportHub, "export-hub", "", "export hub name")
	flags.StringVar(&importHub, "import-hub", "", "import hub name")
	flags.StringVar(&exportService, "export-service", "", "name.namespaces")
	flags.StringVar(&importService, "import-service", "", "name.namespaces")
	flags.StringVar(&tunnelAddress, "tunnel-address", tunnelAddress, "tunnel address")
	flags.BoolVar(&reachable, "reachable", reachable, "whether the tunnel is reachable")
	flags.StringVar(&peerAuthorizedData, "peer-authorized-data", peerAuthorizedData, "peer authorized data")
	flags.StringVar(&peerTunnelAddress, "peer-tunnel-address", peerTunnelAddress, "peer tunnel address")
	flags.StringVar(&bindPort, "bind-port", bindPort, "bind port")
	flags.StringVar(&port, "port", port, "port")
	return cmd
}
