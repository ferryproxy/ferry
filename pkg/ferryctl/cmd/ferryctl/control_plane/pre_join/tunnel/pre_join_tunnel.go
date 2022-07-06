package tunnel

import (
	"fmt"

	"github.com/ferry-proxy/ferry/pkg/ferryctl/control_plane"
	"github.com/ferry-proxy/ferry/pkg/ferryctl/kubectl"
	"github.com/ferry-proxy/ferry/pkg/ferryctl/log"
	"github.com/ferry-proxy/ferry/pkg/ferryctl/utils"
	"github.com/ferry-proxy/ferry/pkg/ferryctl/vars"
	"github.com/spf13/cobra"
)

func NewCommand(logger log.Logger) *cobra.Command {
	var (
		controlPlaneTunnelAddress = vars.AutoPlaceholders
		dataPlaneTunnelAddress    = vars.AutoPlaceholders
	)

	cmd := &cobra.Command{
		Use: "tunnel <data-plane-name>",
		Aliases: []string{
			"t",
		},
		Short: "Control plane can't touch the data plane",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("must have cluster name")
			}
			dataPlaneName := args[0]

			if len(args) > 1 {
				return fmt.Errorf("too many arguments")
			}

			if dataPlaneTunnelAddress == vars.AutoPlaceholders {
				kctl := kubectl.NewKubectl()
				if controlPlaneTunnelAddress == vars.AutoPlaceholders {
					address, err := kctl.GetTunnelAddress(cmd.Context())
					if err != nil {
						return err
					}
					controlPlaneTunnelAddress = address
				}

				identity, authorized, err := utils.GetKey()
				if err != nil {
					return err
				}

				err = control_plane.ShowJoinWithTunnel(cmd.Context(), control_plane.ShowJoinWithTunnelConfig{
					ControlPlaneName:          vars.ControlPlaneName,
					ControlPlaneTunnelAddress: controlPlaneTunnelAddress,
					DataPlaneName:             dataPlaneName,
					DataPlaneIdentity:         identity,
					DataPlaneAuthorized:       authorized,
					DataPlaneHostkey:          identity,
				})
				if err != nil {
					return err
				}

			} else {
				dataPlaneApiserverAddress := fmt.Sprintf("%s-apiserver.ferry-tunnel-system.svc:443", dataPlaneName)
				identity, authorized, err := utils.GetKey()
				if err != nil {
					return err
				}

				err = control_plane.ShowJoinWithTunnelForDataPlane(cmd.Context(), control_plane.ShowJoinWithTunnelForDataPlaneConfig{
					DataPlaneName:             dataPlaneName,
					DataPlaneTunnelAddress:    dataPlaneTunnelAddress,
					DataPlaneApiserverAddress: dataPlaneApiserverAddress,
					DataPlaneIdentity:         identity,
					DataPlaneAuthorized:       authorized,
					DataPlaneHostkey:          identity,
				})
				if err != nil {
					return err
				}
			}
			return nil
		},
	}
	flags := cmd.Flags()
	flags.StringVar(&controlPlaneTunnelAddress, "control-plane-tunnel-address", controlPlaneTunnelAddress, "Tunnel address of the control plane connected to another cluster")
	flags.StringVar(&dataPlaneTunnelAddress, "data-plane-tunnel-address", dataPlaneTunnelAddress, "Tunnel address of the data plane connected to another cluster")

	return cmd
}
