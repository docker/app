package commands

import (
	"fmt"

	"github.com/docker/app/internal"
	"github.com/docker/app/internal/packager"
	"github.com/docker/cli/cli/command"
	"github.com/spf13/cobra"
)

func versionCmd(dockerCli command.Cli) *cobra.Command {
	var onlyBaseImage bool
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Long: `Print version information

The --base-invocation-image will return the base invocation image name only. This can be useful for

	docker pull $(docker app version --base-invocation-image)
	
In order to be able to build an invocation images when using docker app from an offline system.
		`,
		Run: func(cmd *cobra.Command, args []string) {
			image := packager.BaseInvocationImage(dockerCli)
			if onlyBaseImage {
				fmt.Fprintln(dockerCli.Out(), image)
			} else {
				fmt.Fprintln(dockerCli.Out(), internal.FullVersion(image))
			}
		},
	}
	cmd.Flags().BoolVar(&onlyBaseImage, "base-invocation-image", false, "Print CNAB base invocation image to be used")
	return cmd
}
