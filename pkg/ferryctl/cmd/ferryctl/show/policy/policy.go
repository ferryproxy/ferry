package policy

import (
	"github.com/ferry-proxy/ferry/pkg/ferryctl/kubectl"
	"github.com/ferry-proxy/ferry/pkg/ferryctl/log"
	"github.com/spf13/cobra"
)

var example = "kubectl get routepolicy.traffic.ferry.zsm.io -n ferry-system\n"

func NewCommand(logger log.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use: "route-policy",
		Aliases: []string{
			"policy",
			"p",
		},
		Short:   "Show route policy",
		Example: example,
		RunE: func(cmd *cobra.Command, args []string) error {
			kctl := kubectl.NewKubectl()
			return kctl.Wrap(cmd.Context(), "get", "routepolicy.traffic.ferry.zsm.io", "-n", "ferry-system")
		},
	}
	return cmd
}
