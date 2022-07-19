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

package forward

import (
	"fmt"

	"github.com/ferryproxy/ferry/pkg/ferryctl/cmd/ferryctl/local/forward/dial"
	"github.com/ferryproxy/ferry/pkg/ferryctl/cmd/ferryctl/local/forward/listen"
	"github.com/ferryproxy/ferry/pkg/ferryctl/log"
	"github.com/spf13/cobra"
)

func NewCommand(logger log.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Args: cobra.NoArgs,
		Use:  "forward <command>",
		Aliases: []string{
			"f",
		},
		Short: "local forward commands",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("subcommand is required")
		},
	}
	cmd.AddCommand(
		dial.NewCommand(logger),
		listen.NewCommand(logger),
	)
	return cmd
}
