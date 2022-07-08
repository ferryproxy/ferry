package control_plane

import (
	"fmt"

	initcmd "github.com/ferry-proxy/ferry/pkg/ferryctl/cmd/ferryctl/control_plane/init"
	"github.com/ferry-proxy/ferry/pkg/ferryctl/cmd/ferryctl/control_plane/join"
	"github.com/ferry-proxy/ferry/pkg/ferryctl/log"
	"github.com/spf13/cobra"
)

func NewCommand(logger log.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Args: cobra.NoArgs,
		Use:  "control-plane",
		Aliases: []string{
			"control",
			"c",
		},
		Short: "Control plane commands",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("subcommand is required")
		},
	}

	cmd.AddCommand(
		initcmd.NewCommand(logger),
		join.NewCommand(logger),
	)
	return cmd
}
