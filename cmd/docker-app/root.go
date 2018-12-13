package main

import (
	"github.com/docker/app/internal"
	"github.com/docker/cli/cli/command"
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
// FIXME(vdemeester) use command.Cli interface
func newRootCmd(dockerCli command.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "app",
		Short: "Docker Application Packages",
		Long:  `Build and deploy Docker Application Packages.`,
	}
	addCommands(cmd, dockerCli)
	return cmd
}

// addCommands adds all the commands from cli/command to the root command
func addCommands(cmd *cobra.Command, dockerCli command.Cli) {
	cmd.AddCommand(
		deployCmd(dockerCli),
		initCmd(),
		inspectCmd(dockerCli),
		mergeCmd(dockerCli),
		pushCmd(),
		renderCmd(dockerCli),
		splitCmd(),
		validateCmd(),
		versionCmd(dockerCli),
		completionCmd(dockerCli, cmd),
	)
	if internal.Experimental == "on" {
		cmd.AddCommand(
			imageAddCmd(),
			imageLoadCmd(),
			packCmd(dockerCli),
			pullCmd(),
			unpackCmd(),
		)
	}
}

func firstOrEmpty(list []string) string {
	if len(list) != 0 {
		return list[0]
	}
	return ""
}
