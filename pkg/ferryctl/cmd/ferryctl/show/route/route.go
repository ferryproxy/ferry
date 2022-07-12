package route

import (
	"strings"

	"github.com/ferryproxy/ferry/pkg/consts"
	"github.com/ferryproxy/ferry/pkg/ferryctl/kubectl"
	"github.com/ferryproxy/ferry/pkg/ferryctl/log"
	"github.com/spf13/cobra"
)

var example = []string{"get", "route.traffic.ferryproxy.io", "-n", consts.FerryNamespace}

func NewCommand(logger log.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use: "route",
		Aliases: []string{
			"r",
		},
		Short:   "Show route",
		Example: "kubectl " + strings.Join(example, " "),
		RunE: func(cmd *cobra.Command, args []string) error {
			kctl := kubectl.NewKubectl()
			return kctl.Wrap(cmd.Context(), example...)
		},
	}
	return cmd
}
