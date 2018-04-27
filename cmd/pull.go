package cmd

import (
	"github.com/docker/cli/cli"
	"github.com/docker/lunchbox/internal"
	"github.com/docker/lunchbox/packager"
	"github.com/spf13/cobra"
)

var pullCmd = &cobra.Command{
	Use:   "pull <repotag>",
	Short: "Pull an app from a registry",
	Args:  cli.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return packager.Pull(args[0])
	},
}

func init() {
	if internal.Experimental == "on" {
		rootCmd.AddCommand(pullCmd)
	}
}
