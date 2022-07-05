package tunnel

import (
	"fmt"

	"github.com/ferry-proxy/ferry/pkg/ferryctl/data_plane"
	"github.com/ferry-proxy/ferry/pkg/ferryctl/log"
	"github.com/ferry-proxy/ferry/pkg/ferryctl/vars"
	"github.com/spf13/cobra"
)

func NewCommand(logger log.Logger) *cobra.Command {
	var (
		dataPlaneReachable = true
	)

	cmd := &cobra.Command{
		Use: "tunnel <data-plane-name>",
		Aliases: []string{
			"t",
		},
		Short: "Data plane join tunnel commands",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("must have cluster name")
			}
			name := args[0]
			if len(args) > 1 {
				return fmt.Errorf("too many arguments")
			}

			dataPlaneApiserverAddress := fmt.Sprintf("%s-apiserver.ferry-tunnel-system.svc:443", name)
			err := data_plane.ShowJoinDone(cmd.Context(), data_plane.ShowJoinDoneConfig{
				ControlPlaneName:               vars.ControlPlaneName,
				DataPlaneName:                  name,
				DataPlaneReachable:             dataPlaneReachable,
				DataPlaneApiserverAddress:      dataPlaneApiserverAddress,
				DataPlaneNavigationClusterName: vars.ControlPlaneName,
				DataPlaneReceptionClusterName:  vars.ControlPlaneName,
			})
			if err != nil {
				return err
			}
			return nil
		},
	}
	flags := cmd.Flags()
	flags.BoolVar(&dataPlaneReachable, "data-plane-reachable", dataPlaneReachable, "Whether the data plane is reachable")
	return cmd
}
