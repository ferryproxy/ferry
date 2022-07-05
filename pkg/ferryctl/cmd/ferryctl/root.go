package ferryctl

import (
	"fmt"

	"github.com/ferry-proxy/ferry/pkg/ferryctl/cmd/ferryctl/control_plane"
	"github.com/ferry-proxy/ferry/pkg/ferryctl/cmd/ferryctl/data_plane"
	"github.com/ferry-proxy/ferry/pkg/ferryctl/cmd/ferryctl/local"
	"github.com/ferry-proxy/ferry/pkg/ferryctl/cmd/ferryctl/show"
	"github.com/ferry-proxy/ferry/pkg/ferryctl/log"
	"github.com/ferry-proxy/ferry/pkg/ferryctl/vars"
	"github.com/spf13/cobra"
)

// NewCommand returns a new cobra.Command for root
func NewCommand(logger log.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Args:  cobra.NoArgs,
		Use:   "ferryctl",
		Short: "A simple operation and maintenance tool for ferry",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("subcommand is required")
		},
	}

	persistentFlags := cmd.PersistentFlags()
	persistentFlags.StringVar(&vars.KubeconfigPath, "kubeconfig", vars.KubeconfigPath, "override the default kubeconfig path")
	persistentFlags.StringVar(&vars.FerryControllerImage, "ferry-controller-image", vars.FerryControllerImage, "default ferry controller image")
	persistentFlags.StringVar(&vars.FerryTunnelImage, "ferry-tunnel-image", vars.FerryTunnelImage, "default ferry tunnel image")
	persistentFlags.StringVar(&vars.ControlPlaneName, "control-plane-name", vars.ControlPlaneName, "default control plane name")

	cmd.AddCommand(
		control_plane.NewCommand(logger),
		data_plane.NewCommand(logger),
		local.NewCommand(logger),
		show.NewCommand(logger),
	)
	return cmd
}
