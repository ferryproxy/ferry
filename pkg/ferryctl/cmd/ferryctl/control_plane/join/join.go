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
	"strings"

	import_cmd "github.com/ferryproxy/ferry/pkg/ferryctl/cmd/ferryctl/local/manual/import"
	"github.com/ferryproxy/ferry/pkg/ferryctl/control_plane"
	"github.com/ferryproxy/ferry/pkg/ferryctl/kubectl"
	"github.com/ferryproxy/ferry/pkg/ferryctl/log"
	"github.com/ferryproxy/ferry/pkg/ferryctl/utils"
	"github.com/ferryproxy/ferry/pkg/ferryctl/vars"
	"github.com/spf13/cobra"
)

func NewCommand(logger log.Logger) *cobra.Command {
	var (
		controlPlaneTunnelAddress = vars.AutoPlaceholders
		dataPlaneTunnelAddress    = vars.AutoPlaceholders
		dataPlaneApiserverAddress = vars.AutoPlaceholders
		controlPlaneReachable     = true
		dataPlaneReachable        = true
		dataPlaneNavigationWay    = []string{}
		dataPlaneReceptionWay     = []string{}
		dataPlaneNavigationProxy  = []string{}
		dataPlaneReceptionProxy   = []string{}
	)

	cmd := &cobra.Command{
		Args: cobra.ExactArgs(1),
		Use:  "join <data-plane-hub-name>",
		Aliases: []string{
			"j",
		},
		Short: "Control plane join commands",
		Long:  `Control plane join commands is used to join other data plane`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			dataPlaneName := args[0]

			if !controlPlaneReachable {
				controlPlaneTunnelAddress = ""
			}

			if !dataPlaneReachable {
				dataPlaneTunnelAddress = ""
			}

			kctl := kubectl.NewKubectl()
			if controlPlaneTunnelAddress == vars.ControlPlaneName {
				controlPlaneTunnelAddress, err = kctl.GetTunnelAddress(cmd.Context())
				if err != nil {
					return err
				}
			}

			fargs := []string{}

			if !controlPlaneReachable {
				if !dataPlaneReachable {
					return fmt.Errorf("TODO: data plane and control plane is not reachable")
				}

				fargs = []string{
					"--reachable=false",
					"--peer-tunnel-address=" + dataPlaneTunnelAddress,
					"--export-service=kubernetes.default",
					"--port=443",
					"--import-service=" + dataPlaneName + "-apiserver.ferry-tunnel-system",
					"--import-hub=" + vars.ControlPlaneName + "-" + dataPlaneName + "-apiserver",
					"--export-hub=" + dataPlaneName + "-apiserver",
					"--route-name=" + dataPlaneName,
				}
				dataPlaneApiserverAddress = dataPlaneName + "-apiserver.ferry-tunnel-system:443"
			} else {
				if !dataPlaneReachable {
					fargs = []string{
						"--reachable=true",
						"--tunnel-address=" + controlPlaneTunnelAddress,
						"--export-service=kubernetes.default",
						"--port=443",
						"--import-service=" + dataPlaneName + "-apiserver.ferry-tunnel-system",
						"--import-hub=" + vars.ControlPlaneName + "-" + dataPlaneName + "-apiserver",
						"--export-hub=" + dataPlaneName + "-apiserver",
						"--route-name=" + dataPlaneName,
					}
					dataPlaneApiserverAddress = dataPlaneName + "-apiserver.ferry-tunnel-system:443"
				}
			}

			if len(fargs) != 0 {
				fmt.Printf("# > ferryctl local manual import %s\n", strings.Join(fargs, " "))
				sub := import_cmd.NewCommand(logger)
				sub.SetArgs(fargs)
				err := sub.ExecuteContext(cmd.Context())
				if err != nil {
					return err
				}
			}

			next, err := control_plane.ShowJoin(cmd.Context(), control_plane.ShowJoinConfig{
				ControlPlaneName:          vars.ControlPlaneName,
				DataPlaneName:             dataPlaneName,
				DataPlaneApiserverAddress: dataPlaneApiserverAddress,
				DataPlaneTunnelAddress:    dataPlaneTunnelAddress,
				DataPlaneReachable:        dataPlaneReachable,
				ControlPlaneTunnelAddress: controlPlaneTunnelAddress,
				ControlPlaneReachable:     controlPlaneReachable,
				DataPlaneNavigationWay:    dataPlaneNavigationWay,
				DataPlaneReceptionWay:     dataPlaneReceptionWay,
				DataPlaneNavigationProxy:  dataPlaneNavigationProxy,
				DataPlaneReceptionProxy:   dataPlaneReceptionProxy,
			})
			if err != nil {
				return err
			}

			utils.Prompt(
				fmt.Sprintf("join the %s data cluster", dataPlaneName),
				next,
			)

			return nil
		},
	}
	flags := cmd.Flags()
	flags.StringVar(&controlPlaneTunnelAddress, "control-plane-tunnel-address", controlPlaneTunnelAddress, "Tunnel address of the control plane connected to another cluster")
	flags.StringVar(&dataPlaneTunnelAddress, "data-plane-tunnel-address", dataPlaneTunnelAddress, "Tunnel address of the data plane connected to another cluster")
	flags.StringVar(&dataPlaneApiserverAddress, "data-plane-apiserver-address", dataPlaneApiserverAddress, "Apiserver address of the data plane for control plane")
	flags.BoolVar(&controlPlaneReachable, "control-plane-reachable", controlPlaneReachable, "Control plane is reachable")
	flags.BoolVar(&dataPlaneReachable, "data-plane-reachable", dataPlaneReachable, "Data plane is reachable")
	flags.StringSliceVar(&dataPlaneNavigationWay, "data-plane-navigation-way", dataPlaneNavigationWay, "Navigation hub name of the data plane connected to another cluster")
	flags.StringSliceVar(&dataPlaneReceptionWay, "data-plane-reception-way", dataPlaneReceptionWay, "Reception hub name of the data plane connected to another cluster")
	flags.StringSliceVar(&dataPlaneNavigationProxy, "data-plane-navigation-proxy", dataPlaneNavigationProxy, "Navigation proxy name of the data plane connected to another cluster")
	flags.StringSliceVar(&dataPlaneReceptionProxy, "data-plane-reception-proxy", dataPlaneReceptionProxy, "Reception proxy name of the data plane connected to another cluster")
	return cmd
}
