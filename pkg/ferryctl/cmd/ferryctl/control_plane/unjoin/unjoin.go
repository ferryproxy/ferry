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

package unjoin

import (
	"github.com/ferryproxy/ferry/pkg/consts"
	"github.com/ferryproxy/ferry/pkg/ferryctl/kubectl"
	"github.com/ferryproxy/ferry/pkg/ferryctl/log"
	"github.com/spf13/cobra"
)

func NewCommand(logger log.Logger) *cobra.Command {

	cmd := &cobra.Command{
		Args: cobra.ExactArgs(1),
		Use:  "unjoin <data-plane-hub-name>",
		Aliases: []string{
			"u",
		},
		Short: "Control plane unjoin commands",
		Long:  `Control plane unjoin commands`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			dataPlaneName := args[0]

			kctl := kubectl.NewKubectl()
			kctl.Wrap(cmd.Context(), "delete", "hub.traffic.ferryproxy.io", "-n", consts.FerryNamespace, dataPlaneName)
			kctl.Wrap(cmd.Context(), "delete", "secret", "-n", consts.FerryNamespace, dataPlaneName)
			kctl.Wrap(cmd.Context(), "delete", "cm", "-n", consts.FerryTunnelNamespace, "-l", consts.TunnelRouteKey+"="+dataPlaneName)

			return nil
		},
	}
	return cmd
}
