package cluster_information

import (
	"github.com/ferry-proxy/ferry/pkg/ferryctl/kubectl"
	"github.com/ferry-proxy/ferry/pkg/ferryctl/log"
	"github.com/spf13/cobra"
)

var example = "kubectl get clusterinformations.ferry.zsm.io -n ferry-system\n"

func NewCommand(logger log.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use: "cluster-information",
		Aliases: []string{
			"cluster",
			"ci",
		},
		Short:   "Show cluster information",
		Example: example,
		RunE: func(cmd *cobra.Command, args []string) error {
			kctl := kubectl.NewKubectl()
			return kctl.Wrap(cmd.Context(), "get", "clusterinformations.ferry.zsm.io", "-n", "ferry-system")
		},
	}
	return cmd
}
