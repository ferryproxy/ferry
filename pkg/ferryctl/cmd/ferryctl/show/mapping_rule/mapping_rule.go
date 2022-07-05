package mapping_rule

import (
	"github.com/ferry-proxy/ferry/pkg/ferryctl/kubectl"
	"github.com/ferry-proxy/ferry/pkg/ferryctl/log"
	"github.com/spf13/cobra"
)

var example = "kubectl get mappingrule.ferry.zsm.io -n ferry-system\n"

func NewCommand(logger log.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use: "mapping-rule",
		Aliases: []string{
			"mapping",
			"mr",
		},
		Short:   "Show mapping rule",
		Example: example,
		RunE: func(cmd *cobra.Command, args []string) error {
			kctl := kubectl.NewKubectl()
			return kctl.Wrap(cmd.Context(), "get", "mappingrule.ferry.zsm.io", "-n", "ferry-system")
		},
	}
	return cmd
}
