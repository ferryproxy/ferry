package tunnel

import (
	"github.com/ferry-proxy/ferry/pkg/ferryctl/kubectl"
	"github.com/ferry-proxy/ferry/pkg/ferryctl/log"
	"github.com/spf13/cobra"
)

var example = "kubectl exec deploy/ferry-tunnel -n ferry-tunnel-system -- cat bridge.conf\n"

func NewCommand(logger log.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use: "tunnel",
		Aliases: []string{
			"t",
		},
		Short:   "Tunnel rules",
		Example: example,
		RunE: func(cmd *cobra.Command, args []string) error {
			kctl := kubectl.NewKubectl()
			return kctl.Wrap(cmd.Context(), "exec", "deploy/ferry-tunnel", "-n", "ferry-tunnel-system", "--", "cat", "bridge.conf")
		},
	}
	return cmd
}
