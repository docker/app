package main

import (
	"github.com/docker/app/renderer"
	"github.com/docker/cli/cli"
	"github.com/spf13/cobra"
)

// inspectCmd represents the inspect command
func inspectCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "inspect [<app-name>]",
		Short: "Shows metadata and settings for a given application",
		Args:  cli.RequiresMaxArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return renderer.Inspect(firstOrEmpty(args))
		},
	}
}
