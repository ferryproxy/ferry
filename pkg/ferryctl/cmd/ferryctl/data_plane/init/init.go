package init

import (
	"github.com/ferry-proxy/ferry/pkg/ferryctl/data_plane"
	"github.com/ferry-proxy/ferry/pkg/ferryctl/log"
	"github.com/ferry-proxy/ferry/pkg/ferryctl/vars"
	"github.com/spf13/cobra"
)

func NewCommand(logger log.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:  "init",
		Args: cobra.NoArgs,
		Aliases: []string{
			"i",
		},
		Short: "Data plane init commands",
		Long:  `Data plane init commands is used to initialize the data plane`,
		RunE: func(cmd *cobra.Command, args []string) error {
			err := data_plane.ClusterInit(cmd.Context(), data_plane.ClusterInitConfig{
				FerryTunnelImage: vars.FerryTunnelImage,
			})
			if err != nil {
				return err
			}
			return nil
		},
	}
	return cmd
}
