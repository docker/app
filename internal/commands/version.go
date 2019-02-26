package commands

import (
	"fmt"

	"github.com/docker/app/internal"
	"github.com/docker/cli/cli/command"
	"github.com/spf13/cobra"
)

func versionCmd(dockerCli command.Cli) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintln(dockerCli.Out(), internal.FullVersion())
		},
	}
}
