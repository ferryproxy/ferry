package hub

import (
	"github.com/ferry-proxy/ferry/pkg/ferryctl/kubectl"
	"github.com/ferry-proxy/ferry/pkg/ferryctl/log"
	"github.com/spf13/cobra"
)

var example = "kubectl get hub.traffic.ferry.zsm.io -n ferry-system\n"

func NewCommand(logger log.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use: "hub",
		Aliases: []string{
			"h",
		},
		Short:   "Show hub",
		Example: example,
		RunE: func(cmd *cobra.Command, args []string) error {
			kctl := kubectl.NewKubectl()
			return kctl.Wrap(cmd.Context(), "get", "hub.traffic.ferry.zsm.io", "-n", "ferry-system")
		},
	}
	return cmd
}
