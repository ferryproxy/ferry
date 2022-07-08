package hub

import (
	"strings"

	"github.com/ferry-proxy/ferry/pkg/consts"
	"github.com/ferry-proxy/ferry/pkg/ferryctl/kubectl"
	"github.com/ferry-proxy/ferry/pkg/ferryctl/log"
	"github.com/spf13/cobra"
)

var example = []string{"get", "hub.traffic.ferry.zsm.io", "-n", consts.FerryNamespace}

func NewCommand(logger log.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use: "hub",
		Aliases: []string{
			"h",
		},
		Short:   "Show hub",
		Example: "kubectl " + strings.Join(example, " "),
		RunE: func(cmd *cobra.Command, args []string) error {
			kctl := kubectl.NewKubectl()
			return kctl.Wrap(cmd.Context(), example...)
		},
	}
	return cmd
}
