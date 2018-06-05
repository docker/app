package main

import (
	"github.com/docker/app/packager"
	"github.com/docker/cli/cli"
	"github.com/spf13/cobra"
)

func loadCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "load <repotag>",
		Short: "Load an application from a docker image",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return packager.Load(args[0], ".")
		},
	}
}
