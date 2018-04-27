package cmd

import (
	"github.com/docker/lunchbox/internal"
	"github.com/docker/lunchbox/packager"
	"github.com/spf13/cobra"
)

var loadCmd = &cobra.Command{
	Use:   "load <repotag>",
	Short: "Load an app from docker",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		app := ""
		if len(args) > 0 {
			app = args[0]
		}
		return packager.Load(app)
	},
}

func init() {
	if internal.Experimental == "on" {
		rootCmd.AddCommand(loadCmd)
	}
}
