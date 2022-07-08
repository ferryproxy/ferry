package policy

import (
	"strings"

	"github.com/ferry-proxy/ferry/pkg/consts"
	"github.com/ferry-proxy/ferry/pkg/ferryctl/kubectl"
	"github.com/ferry-proxy/ferry/pkg/ferryctl/log"
	"github.com/spf13/cobra"
)

var example = []string{"get", "routepolicy.traffic.ferry.zsm.io", "-n", consts.FerryNamespace}

func NewCommand(logger log.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use: "route-policy",
		Aliases: []string{
			"policy",
			"p",
		},
		Short:   "Show route policy",
		Example: "kubectl " + strings.Join(example, " "),
		RunE: func(cmd *cobra.Command, args []string) error {
			kctl := kubectl.NewKubectl()
			return kctl.Wrap(cmd.Context(), example...)
		},
	}
	return cmd
}
