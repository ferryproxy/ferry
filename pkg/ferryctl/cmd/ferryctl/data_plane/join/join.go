package join

import (
	"fmt"

	"github.com/ferry-proxy/ferry/pkg/ferryctl/data_plane"
	"github.com/ferry-proxy/ferry/pkg/ferryctl/kubectl"
	"github.com/ferry-proxy/ferry/pkg/ferryctl/log"
	"github.com/ferry-proxy/ferry/pkg/ferryctl/utils"
	"github.com/ferry-proxy/ferry/pkg/ferryctl/vars"
	"github.com/spf13/cobra"
)

func NewCommand(logger log.Logger) *cobra.Command {
	var (
		controlPlaneHubName        = vars.ControlPlaneName
		dataPlaneTunnelAddress     = vars.AutoPlaceholders
		dataPlaneApiserverAddress  = vars.AutoPlaceholders
		dataPlaneReachable         = true
		dataPlaneNavigationHubName = []string{}
		dataPlaneReceptionHubName  = []string{}
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
				ControlPlaneName:           controlPlaneHubName,
				DataPlaneName:              name,
				DataPlaneReachable:         dataPlaneReachable,
				DataPlaneApiserverAddress:  dataPlaneApiserverAddress,
				DataPlaneTunnelAddress:     dataPlaneTunnelAddress,
				DataPlaneNavigationHubName: dataPlaneNavigationHubName,
				DataPlaneReceptionHubName:  dataPlaneNavigationHubName,
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
	flags.StringSliceVar(&dataPlaneNavigationHubName, "data-plane-navigation-hub-name", dataPlaneNavigationHubName, "Navigation hub name of the data plane connected to another cluster")
	flags.StringSliceVar(&dataPlaneReceptionHubName, "data-plane-reception-hub-name", dataPlaneReceptionHubName, "Reception hub name of the data plane connected to another cluster")
	return cmd
}
