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

package init

import (
	"github.com/ferryproxy/ferry/pkg/ferryctl/data_plane"
	"github.com/ferryproxy/ferry/pkg/ferryctl/log"
	"github.com/ferryproxy/ferry/pkg/ferryctl/vars"
	"github.com/spf13/cobra"
)

func NewCommand(logger log.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:  "init",
		Args: cobra.NoArgs,
		Aliases: []string{
			"i",
		},
		Short: "Data plane init commands",
		Long:  `Data plane init commands is used to initialize the data plane`,
		RunE: func(cmd *cobra.Command, args []string) error {
			err := data_plane.ClusterInit(cmd.Context(), data_plane.ClusterInitConfig{
				FerryTunnelImage: vars.FerryTunnelImage,
			})
			if err != nil {
				return err
			}
			return nil
		},
	}
	return cmd
}
