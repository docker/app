package main

import (
	"github.com/docker/cli/cli"
	"github.com/docker/lunchbox/packager"
	"github.com/spf13/cobra"
)

var pullCmd = &cobra.Command{
	Use:   "pull <repotag>",
	Short: "Pull an application from a registry",
	Args:  cli.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return packager.Pull(args[0])
	},
}

func init() {
	rootCmd.AddCommand(pullCmd)
}
