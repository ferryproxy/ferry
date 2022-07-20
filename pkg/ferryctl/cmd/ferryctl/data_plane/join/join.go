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

package join

import (
	"fmt"

	"github.com/ferryproxy/ferry/pkg/ferryctl/data_plane"
	"github.com/ferryproxy/ferry/pkg/ferryctl/kubectl"
	"github.com/ferryproxy/ferry/pkg/ferryctl/log"
	"github.com/ferryproxy/ferry/pkg/ferryctl/utils"
	"github.com/ferryproxy/ferry/pkg/ferryctl/vars"
	"github.com/spf13/cobra"
)

func NewCommand(logger log.Logger) *cobra.Command {
	var (
		controlPlaneHubName       = vars.ControlPlaneName
		dataPlaneTunnelAddress    = vars.AutoPlaceholders
		dataPlaneApiserverAddress = vars.AutoPlaceholders
		dataPlaneReachable        = true
		dataPlaneNavigationWay    = []string{}
		dataPlaneReceptionWay     = []string{}
		dataPlaneNavigationProxy  = []string{}
		dataPlaneReceptionProxy   = []string{}
	)

	cmd := &cobra.Command{
		Use:  "join <data-plane-hub-name>",
		Args: cobra.ExactArgs(1),
		Aliases: []string{
			"j",
		},
		Short: "Data plane join commands",
		Long:  `Data plane join commands is used to join itself to control plane`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			name := args[0]

			kctl := kubectl.NewKubectl()

			if dataPlaneReachable {
				if dataPlaneTunnelAddress == vars.AutoPlaceholders {
					dataPlaneTunnelAddress, err = kctl.GetTunnelAddress(cmd.Context())
					if err != nil {
						return err
					}
				}
			} else {
				dataPlaneTunnelAddress = ""
			}

			if dataPlaneApiserverAddress == vars.AutoPlaceholders {
				dataPlaneApiserverAddress, err = kctl.GetApiserverAddress(cmd.Context())
				if err != nil {
					return err
				}
			}

			next, err := data_plane.ShowJoinDone(cmd.Context(), data_plane.ShowJoinDoneConfig{
				ControlPlaneName:          controlPlaneHubName,
				DataPlaneName:             name,
				DataPlaneReachable:        dataPlaneReachable,
				DataPlaneApiserverAddress: dataPlaneApiserverAddress,
				DataPlaneTunnelAddress:    dataPlaneTunnelAddress,
				DataPlaneNavigationWay:    dataPlaneNavigationWay,
				DataPlaneReceptionWay:     dataPlaneReceptionWay,
				DataPlaneNavigationProxy:  dataPlaneNavigationProxy,
				DataPlaneReceptionProxy:   dataPlaneReceptionProxy,
			})
			if err != nil {
				return err
			}

			utils.Prompt(
				fmt.Sprintf("join the %s data cluster", controlPlaneHubName),
				next,
			)
			return nil
		},
	}
	flags := cmd.Flags()
	flags.StringVar(&controlPlaneHubName, "control-plane-hub-name", controlPlaneHubName, "Name of the control plane hub")
	flags.StringVar(&dataPlaneTunnelAddress, "data-plane-tunnel-address", dataPlaneTunnelAddress, "Tunnel address of the data plane connected to another cluster")
	flags.StringVar(&dataPlaneApiserverAddress, "data-plane-apiserver-address", dataPlaneApiserverAddress, "Apiserver address of the data plane for control plane")
	flags.BoolVar(&dataPlaneReachable, "data-plane-reachable", dataPlaneReachable, "Whether the data plane is reachable")
	flags.StringSliceVar(&dataPlaneNavigationWay, "data-plane-navigation-way", dataPlaneNavigationWay, "Navigation hub name of the data plane connected to another cluster")
	flags.StringSliceVar(&dataPlaneReceptionWay, "data-plane-reception-way", dataPlaneReceptionWay, "Reception hub name of the data plane connected to another cluster")
	flags.StringSliceVar(&dataPlaneNavigationProxy, "data-plane-navigation-proxy", dataPlaneNavigationProxy, "Navigation proxy name of the data plane connected to another cluster")
	flags.StringSliceVar(&dataPlaneReceptionProxy, "data-plane-reception-proxy", dataPlaneReceptionProxy, "Reception proxy name of the data plane connected to another cluster")
	return cmd
}
