package image

import (
	"github.com/docker/app/internal/cliopts"
	"github.com/docker/cli/cli/command"
	"github.com/spf13/cobra"
)

// Cmd is the image top level command
func Cmd(dockerCli command.Cli, installerContext *cliopts.InstallerContextOptions) *cobra.Command {
	cmd := &cobra.Command{
		Short: "Manage App images",
		Use:   "image",
	}

	cmd.AddCommand(
		listCmd(dockerCli),
		rmCmd(),
		tagCmd(),
		inspectCmd(dockerCli, installerContext),
		renderCmd(dockerCli, installerContext),
	)

	return cmd
}
