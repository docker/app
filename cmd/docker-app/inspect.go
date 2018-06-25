package main

import (
	"github.com/docker/app/internal/packager"
	"github.com/docker/app/internal/renderer"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/spf13/cobra"
)

// inspectCmd represents the inspect command
func inspectCmd(dockerCli command.Cli) *cobra.Command {
	return &cobra.Command{
		Use:   "inspect [<app-name>]",
		Short: "Shows metadata and settings for a given application",
		Args:  cli.RequiresMaxArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			appname, cleanup, err := packager.Extract(firstOrEmpty(args))
			if err != nil {
				return err
			}
			defer cleanup()
			return renderer.Inspect(dockerCli.Out(), appname)
		},
	}
}
