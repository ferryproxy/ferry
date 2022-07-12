package tunnel

import (
	"strings"

	"github.com/ferryproxy/ferry/pkg/consts"
	"github.com/ferryproxy/ferry/pkg/ferryctl/kubectl"
	"github.com/ferryproxy/ferry/pkg/ferryctl/log"
	"github.com/spf13/cobra"
)

var example = []string{"exec", "deploy/" + consts.FerryTunnelName, "-n", consts.FerryTunnelNamespace, "--", "cat", "bridge.conf"}

func NewCommand(logger log.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use: "tunnel",
		Aliases: []string{
			"t",
		},
		Short:   "Tunnel rules",
		Example: "kubectl " + strings.Join(example, " "),
		RunE: func(cmd *cobra.Command, args []string) error {
			kctl := kubectl.NewKubectl()
			return kctl.Wrap(cmd.Context(), example...)
		},
	}
	return cmd
}
