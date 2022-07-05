package show

import (
	"fmt"

	"github.com/ferry-proxy/ferry/pkg/ferryctl/cmd/ferryctl/show/cluster_information"
	"github.com/ferry-proxy/ferry/pkg/ferryctl/cmd/ferryctl/show/mapping_rule"
	"github.com/ferry-proxy/ferry/pkg/ferryctl/cmd/ferryctl/show/policy"
	"github.com/ferry-proxy/ferry/pkg/ferryctl/cmd/ferryctl/show/tunnel"
	"github.com/ferry-proxy/ferry/pkg/ferryctl/log"
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
		cluster_information.NewCommand(logger),
		mapping_rule.NewCommand(logger),
		policy.NewCommand(logger),
		tunnel.NewCommand(logger),
	)
	return cmd
}
