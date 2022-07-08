package manual

import (
	"fmt"

	"github.com/ferry-proxy/ferry/pkg/ferryctl/cmd/ferryctl/local/manual/export"
	import_cmd "github.com/ferry-proxy/ferry/pkg/ferryctl/cmd/ferryctl/local/manual/import"
	"github.com/ferry-proxy/ferry/pkg/ferryctl/log"
	"github.com/spf13/cobra"
)

func NewCommand(logger log.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Args: cobra.NoArgs,
		Use:  "manual",
		Aliases: []string{
			"m",
		},
		Short: "manual commands",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("subcommand is required")
		},
	}
	cmd.AddCommand(
		export.NewCommand(logger),
		import_cmd.NewCommand(logger),
	)
	return cmd
}
