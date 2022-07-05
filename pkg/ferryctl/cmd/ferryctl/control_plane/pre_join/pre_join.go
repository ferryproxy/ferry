package pre_join

import (
	"fmt"

	"github.com/ferry-proxy/ferry/pkg/ferryctl/cmd/ferryctl/control_plane/pre_join/direct"
	"github.com/ferry-proxy/ferry/pkg/ferryctl/cmd/ferryctl/control_plane/pre_join/tunnel"
	"github.com/ferry-proxy/ferry/pkg/ferryctl/log"
	"github.com/spf13/cobra"
)

func NewCommand(logger log.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Args: cobra.NoArgs,
		Use:  "pre-join <command>",
		Aliases: []string{
			"p",
			"pj",
			"join",
			"j",
		},
		Short: "Generate command for data plane join",
		Long: `Generate command for data plane join
Need to copy the generated command to run on the data plane.
`,
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
