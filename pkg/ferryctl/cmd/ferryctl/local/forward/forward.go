package forward

import (
	"fmt"

	"github.com/ferry-proxy/ferry/pkg/ferryctl/cmd/ferryctl/local/forward/dial"
	"github.com/ferry-proxy/ferry/pkg/ferryctl/cmd/ferryctl/local/forward/listen"
	"github.com/ferry-proxy/ferry/pkg/ferryctl/log"
	"github.com/spf13/cobra"
)

func NewCommand(logger log.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Args: cobra.NoArgs,
		Use:  "forward <command>",
		Aliases: []string{
			"f",
		},
		Short: "local forward commands",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("subcommand is required")
		},
	}
	cmd.AddCommand(
		dial.NewCommand(logger),
		listen.NewCommand(logger),
	)
	return cmd
}
