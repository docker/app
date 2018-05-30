package main

import (
	"github.com/docker/cli/cli"
	"github.com/docker/lunchbox/packager"
	"github.com/spf13/cobra"
)

func loadCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "load <repotag>",
		Short: "Load an application from a docker image",
		Args:  cli.RequiresMaxArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return packager.Load(firstOrEmpty(args), ".")
		},
	}
}
