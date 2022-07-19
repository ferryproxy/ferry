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

package listen

import (
	"fmt"

	"github.com/ferryproxy/ferry/pkg/ferryctl/local"
	"github.com/ferryproxy/ferry/pkg/ferryctl/log"
	"github.com/spf13/cobra"
)

func NewCommand(logger log.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use: "listen <remote service port> <local address port>",
		Aliases: []string{
			"l",
		},
		Short: "local forward listen commands",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 2 {
				return fmt.Errorf("too few arguments")
			}
			if len(args) > 2 {
				return fmt.Errorf("too many arguments")
			}

			return local.ForwardListen(cmd.Context(), args[0], args[1])
		},
	}
	return cmd
}
