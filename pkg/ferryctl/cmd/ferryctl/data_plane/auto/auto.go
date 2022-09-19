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

package auto

import (
	"strings"

	"github.com/ferryproxy/ferry/pkg/ferryctl/data_plane"
	"github.com/ferryproxy/ferry/pkg/ferryctl/kubectl"
	"github.com/ferryproxy/ferry/pkg/ferryctl/log"
	"github.com/ferryproxy/ferry/pkg/ferryctl/registry/joiner"
	"github.com/ferryproxy/ferry/pkg/ferryctl/vars"
	"github.com/spf13/cobra"
)

func NewCommand(logger log.Logger) *cobra.Command {
	var (
		tunnelServiceType = "NodePort"
		registerBaseURL   = ""
	)
	cmd := &cobra.Command{
		Use:  "auto",
		Args: cobra.ExactArgs(1),
		Aliases: []string{
			"a",
		},
		Short: "Data plane init and join command",
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			err := data_plane.ClusterInit(cmd.Context(), data_plane.ClusterInitConfig{
				FerryTunnelImage:  vars.FerryTunnelImage,
				TunnelServiceType: tunnelServiceType,
			})
			if err != nil {
				return err
			}

			data, err := joiner.BuildInitJoiner(joiner.BuildInitJoinerConfig{
				Image:   vars.FerryJoinerImage,
				BaseURL: registerBaseURL,
				HubName: name,
			})

			kctl := kubectl.NewKubectl()
			err = kctl.ApplyWithReader(cmd.Context(), strings.NewReader(data))
			if err != nil {
				return err
			}

			kctl.LogsJoiner(cmd.Context())
			return nil
		},
	}
	flags := cmd.Flags()
	flags.StringVar(&tunnelServiceType, "tunnel-service-type", tunnelServiceType, "Tunnel service type (LoadBalancer or NodePort)")
	flags.StringVar(&registerBaseURL, "register-url", registerBaseURL, "The url of Register")
	return cmd
}
