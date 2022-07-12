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
