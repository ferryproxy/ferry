package direct

import (
	"fmt"

	"github.com/ferry-proxy/ferry/pkg/ferryctl/data_plane"
	"github.com/ferry-proxy/ferry/pkg/ferryctl/kubectl"
	"github.com/ferry-proxy/ferry/pkg/ferryctl/log"
	"github.com/ferry-proxy/ferry/pkg/ferryctl/vars"
	"github.com/spf13/cobra"
)

func NewCommand(logger log.Logger) *cobra.Command {
	var (
		dataPlaneTunnelAddress    = vars.AutoPlaceholders
		dataPlaneApiserverAddress = vars.AutoPlaceholders
		dataPlaneReachable        = true
	)

	cmd := &cobra.Command{
		Use: "direct <data-plane-name>",
		Aliases: []string{
			"d",
		},
		Short: "Data plane join direct commands",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("must have cluster name")
			}
			name := args[0]

			if len(args) > 1 {
				return fmt.Errorf("too many arguments")
			}

			kctl := kubectl.NewKubectl()

			var err error
			if dataPlaneTunnelAddress == vars.AutoPlaceholders {
				dataPlaneTunnelAddress, err = kctl.GetTunnelAddress(cmd.Context())
				if err != nil {
					return err
				}
			}
			if dataPlaneApiserverAddress == vars.AutoPlaceholders {
				dataPlaneApiserverAddress, err = kctl.GetApiserverAddress(cmd.Context())
				if err != nil {
					return err
				}
			}

			err = data_plane.ShowJoinDone(cmd.Context(), data_plane.ShowJoinDoneConfig{
				ControlPlaneName:          vars.ControlPlaneName,
				DataPlaneName:             name,
				DataPlaneReachable:        dataPlaneReachable,
				DataPlaneApiserverAddress: dataPlaneApiserverAddress,
				DataPlaneTunnelAddress:    dataPlaneTunnelAddress,
			})
			if err != nil {
				return err
			}
			return nil
		},
	}
	flags := cmd.Flags()
	flags.StringVar(&dataPlaneTunnelAddress, "data-plane-tunnel-address", dataPlaneTunnelAddress, "Tunnel address of the data plane connected to another cluster")
	flags.StringVar(&dataPlaneApiserverAddress, "data-plane-apiserver-address", dataPlaneApiserverAddress, "Apiserver address of the data plane for control plane")
	flags.BoolVar(&dataPlaneReachable, "data-plane-reachable", dataPlaneReachable, "Whether the data plane is reachable")
	return cmd
}
