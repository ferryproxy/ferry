package show

import (
	"fmt"

	"github.com/ferryproxy/ferry/pkg/ferryctl/cmd/ferryctl/show/hub"
	"github.com/ferryproxy/ferry/pkg/ferryctl/cmd/ferryctl/show/policy"
	"github.com/ferryproxy/ferry/pkg/ferryctl/cmd/ferryctl/show/route"
	"github.com/ferryproxy/ferry/pkg/ferryctl/cmd/ferryctl/show/tunnel"
	"github.com/ferryproxy/ferry/pkg/ferryctl/log"
	"github.com/spf13/cobra"
)

func NewCommand(logger log.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Args: cobra.NoArgs,
		Use:  "show",
		Aliases: []string{
			"s",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("subcommand is required")
		},
	}
	cmd.AddCommand(
		hub.NewCommand(logger),
		route.NewCommand(logger),
		policy.NewCommand(logger),
		tunnel.NewCommand(logger),
	)
	return cmd
}
