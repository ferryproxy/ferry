package data_plane

import (
	"fmt"

	initcmd "github.com/ferryproxy/ferry/pkg/ferryctl/cmd/ferryctl/data_plane/init"
	"github.com/ferryproxy/ferry/pkg/ferryctl/cmd/ferryctl/data_plane/join"
	"github.com/ferryproxy/ferry/pkg/ferryctl/log"
	"github.com/spf13/cobra"
)

func NewCommand(logger log.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Args: cobra.NoArgs,
		Use:  "data-plane",
		Aliases: []string{
			"data",
			"d",
		},
		Short: "Data plane commands",
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
