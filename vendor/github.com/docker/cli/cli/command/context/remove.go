package context

import (
	"fmt"
	"strings"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/spf13/cobra"
)

func newRemoveCommand(dockerCli command.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "rm <ctx1> [<ctx2>...]",
		Aliases: []string{"remove"},
		Short:   "Remove contexts",
		Args:    cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			currentCtx := strings.ToLower(dockerCli.CurrentContext())
			for _, name := range args {
				if strings.ToLower(name) == currentCtx {
					return fmt.Errorf("%q is the current context", name)
				}
				if err := dockerCli.ContextStore().RemoveContext(name); err != nil {
					return err
				}
			}
			return nil
		},
	}
	return cmd
}
