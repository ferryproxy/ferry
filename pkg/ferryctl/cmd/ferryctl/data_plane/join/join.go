package join

import (
	"fmt"

	"github.com/ferry-proxy/ferry/pkg/ferryctl/cmd/ferryctl/data_plane/join/direct"
	"github.com/ferry-proxy/ferry/pkg/ferryctl/cmd/ferryctl/data_plane/join/tunnel"
	"github.com/ferry-proxy/ferry/pkg/ferryctl/log"
	"github.com/spf13/cobra"
)

func NewCommand(logger log.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Args: cobra.NoArgs,
		Use:  "join <command>",
		Aliases: []string{
			"j",
		},
		Short: "Data plane join commands",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("subcommand is required")
		},
	}
	cmd.AddCommand(
		direct.NewCommand(logger),
		tunnel.NewCommand(logger),
	)
	return cmd
}
