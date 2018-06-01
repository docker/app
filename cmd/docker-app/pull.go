package main

import (
	"github.com/docker/app/packager"
	"github.com/docker/cli/cli"
	"github.com/spf13/cobra"
)

func pullCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "pull <repotag>",
		Short: "Pull an application from a registry",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return packager.Pull(args[0])
		},
	}
}
