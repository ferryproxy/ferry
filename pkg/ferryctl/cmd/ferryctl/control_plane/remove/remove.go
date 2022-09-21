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

package remove

import (
	"github.com/ferryproxy/ferry/pkg/consts"
	"github.com/ferryproxy/ferry/pkg/ferryctl/kubectl"
	"github.com/ferryproxy/ferry/pkg/ferryctl/log"
	"github.com/spf13/cobra"
)

func NewCommand(logger log.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Args: cobra.NoArgs,
		Use:  "remove",
		Aliases: []string{
			"r",
		},
		Short: "Control plane remove commands",
		Long:  `Control plane remove commands`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			kctl := kubectl.NewKubectl()
			kctl.Wrap(cmd.Context(), "delete", "-n", consts.FerryNamespace, "routepolicies.traffic.ferryproxy.io", "--all")
			kctl.Wrap(cmd.Context(), "delete", "-n", consts.FerryNamespace, "routes.traffic.ferryproxy.io", "--all")
			kctl.Wrap(cmd.Context(), "delete", "-n", consts.FerryNamespace, "hubs.traffic.ferryproxy.io", "--all")
			kctl.Wrap(cmd.Context(), "delete", "ns", consts.FerryNamespace)
			kctl.Wrap(cmd.Context(), "delete", "ns", consts.FerryTunnelNamespace)
			kctl.Wrap(cmd.Context(), "delete", "crd", "hubs.traffic.ferryproxy.io")
			kctl.Wrap(cmd.Context(), "delete", "crd", "routepolicies.traffic.ferryproxy.io")
			kctl.Wrap(cmd.Context(), "delete", "crd", "routes.traffic.ferryproxy.io")

			return nil
		},
	}
	return cmd
}
