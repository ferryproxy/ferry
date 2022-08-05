/*
Copyright 2022 FerryProxy Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package ferryctl

import (
	"fmt"

	"github.com/ferryproxy/ferry/pkg/ferryctl/cmd/ferryctl/control_plane"
	"github.com/ferryproxy/ferry/pkg/ferryctl/cmd/ferryctl/data_plane"
	"github.com/ferryproxy/ferry/pkg/ferryctl/cmd/ferryctl/local"
	"github.com/ferryproxy/ferry/pkg/ferryctl/cmd/ferryctl/show"
	"github.com/ferryproxy/ferry/pkg/ferryctl/log"
	"github.com/ferryproxy/ferry/pkg/ferryctl/vars"
	"github.com/spf13/cobra"
)

// NewCommand returns a new cobra.Command for root
func NewCommand(logger log.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Args:  cobra.NoArgs,
		Use:   "ferryctl",
		Short: "A simple operation and maintenance tool for ferry",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("subcommand is required")
		},
	}

	persistentFlags := cmd.PersistentFlags()
	persistentFlags.StringVar(&vars.KubeconfigPath, "kubeconfig", vars.KubeconfigPath, "override the default kubeconfig path")
	persistentFlags.StringVar(&vars.PeerKubeconfigPath, "peer-kubeconfig", vars.PeerKubeconfigPath, "this Kubeconfig specifies the handshake peer for operations that require handshaking")
	persistentFlags.StringVar(&vars.FerryControllerImage, "ferry-controller-image", vars.FerryControllerImage, "default ferry controller image")
	persistentFlags.StringVar(&vars.FerryTunnelImage, "ferry-tunnel-image", vars.FerryTunnelImage, "default ferry tunnel image")
	persistentFlags.StringVar(&vars.ControlPlaneName, "control-plane-name", vars.ControlPlaneName, "default control plane name")

	cmd.AddCommand(
		control_plane.NewCommand(logger),
		data_plane.NewCommand(logger),
		local.NewCommand(logger),
		show.NewCommand(logger),
	)
	return cmd
}
