package init

import (
	"fmt"

	"github.com/ferry-proxy/ferry/pkg/ferryctl/data_plane"
	"github.com/ferry-proxy/ferry/pkg/ferryctl/log"
	"github.com/spf13/cobra"
)

func NewCommand(logger log.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use: "init",
		Aliases: []string{
			"i",
		},
		Short: "Data plane init commands",
		Long:  `Control plane init commands is used to initialize the data plane`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return fmt.Errorf("too many arguments")
			}
			err := data_plane.ClusterInit(cmd.Context())
			if err != nil {
				return err
			}
			return nil
		},
	}
	return cmd
}
