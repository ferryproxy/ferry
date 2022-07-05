package local

import (
	"fmt"

	"github.com/ferry-proxy/ferry/pkg/ferryctl/cmd/ferryctl/local/forward"
	"github.com/ferry-proxy/ferry/pkg/ferryctl/log"
	"github.com/spf13/cobra"
)

func NewCommand(logger log.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Args: cobra.NoArgs,
		Use:  "local",
		Aliases: []string{
			"l",
		},
		Short: "local commands",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("subcommand is required")
		},
	}
	cmd.AddCommand(
		forward.NewCommand(logger),
	)
	return cmd
}
