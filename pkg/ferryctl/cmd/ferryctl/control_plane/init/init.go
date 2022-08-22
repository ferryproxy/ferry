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

package init

import (
	"fmt"

	"github.com/ferryproxy/ferry/pkg/ferryctl/control_plane"
	"github.com/ferryproxy/ferry/pkg/ferryctl/kubectl"
	"github.com/ferryproxy/ferry/pkg/ferryctl/log"
	"github.com/ferryproxy/ferry/pkg/ferryctl/vars"
	"github.com/spf13/cobra"
)

func NewCommand(logger log.Logger) *cobra.Command {
	var (
		controlPlaneTunnelAddress = vars.AutoPlaceholders
		controlPlaneReachable     = true
		tunnelServiceType         = "NodePort"
	)

	cmd := &cobra.Command{
		Use:  "init",
		Args: cobra.NoArgs,
		Aliases: []string{
			"i",
		},
		Short: "Control plane init commands",
		Long:  `Control plane init commands is used to initialize the control plane`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return fmt.Errorf("too many arguments")
			}

			kctl := kubectl.NewKubectl()

			var err error
			if controlPlaneTunnelAddress == vars.AutoPlaceholders {
				controlPlaneTunnelAddress, err = kctl.GetTunnelAddress(cmd.Context())
				if err != nil {
					return err
				}
			}

			err = control_plane.ClusterInit(cmd.Context(), control_plane.ClusterInitConfig{
				ControlPlaneName:          vars.ControlPlaneName,
				ControlPlaneReachable:     controlPlaneReachable,
				ControlPlaneTunnelAddress: controlPlaneTunnelAddress,
				FerryControllerImage:      vars.FerryControllerImage,
			})
			if err != nil {
				return err
			}

			return nil
		},
	}
	flags := cmd.Flags()
	flags.StringVar(&controlPlaneTunnelAddress, "control-plane-tunnel-address", controlPlaneTunnelAddress, "Tunnel address of the control plane connected to another cluster")
	flags.BoolVar(&controlPlaneReachable, "control-plane-reachable", controlPlaneReachable, "Whether the control plane is reachable")
	flags.StringVar(&tunnelServiceType, "tunnel-service-type", tunnelServiceType, "Tunnel service type (LoadBalancer or NodePort)")
	return cmd
}
