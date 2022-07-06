package direct

import (
	"fmt"

	"github.com/ferry-proxy/ferry/pkg/ferryctl/control_plane"
	"github.com/ferry-proxy/ferry/pkg/ferryctl/log"
	"github.com/ferry-proxy/ferry/pkg/ferryctl/utils"
	"github.com/ferry-proxy/ferry/pkg/ferryctl/vars"
	"github.com/spf13/cobra"
)

func NewCommand(logger log.Logger) *cobra.Command {
	var (
		dataPlaneTunnelAddress    = vars.AutoPlaceholders
		dataPlaneApiserverAddress = vars.AutoPlaceholders
	)

	cmd := &cobra.Command{
		Use: "direct <data-plane-name>",
		Aliases: []string{
			"d",
		},
		Short: "Clusters can reach each other",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("must have cluster name")
			}
			dataPlaneName := args[0]

			if len(args) > 1 {
				return fmt.Errorf("too many arguments")
			}

			identity, authorized, err := utils.GetKey()
			if err != nil {
				return err
			}

			err = control_plane.ShowJoinWithDirect(cmd.Context(), control_plane.ShowJoinWithDirectConfig{
				DataPlaneName:             dataPlaneName,
				DataPlaneApiserverAddress: dataPlaneApiserverAddress,
				DataPlaneTunnelAddress:    dataPlaneTunnelAddress,
				DataPlaneIdentity:         identity,
				DataPlaneAuthorized:       authorized,
				DataPlaneHostkey:          identity,
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

	return cmd
}
