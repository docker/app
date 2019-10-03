package image

import (
	"github.com/docker/cli/cli/command"
	"github.com/spf13/cobra"
)

// Cmd is the image top level command
func Cmd(dockerCli command.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Short: "Manage application images",
		Use:   "image",
	}

	cmd.AddCommand(
		listCmd(dockerCli),
		rmCmd(dockerCli),
		tagCmd(),
	)

	return cmd
}
