package cmd

import (
	"fmt"
	"os"

	"github.com/docker/lunchbox/internal"
	"github.com/docker/lunchbox/packager"
	"github.com/spf13/cobra"
)

var loadCmd = &cobra.Command{
	Use:   "load <repotag>",
	Short: "Load an app from docker",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		app := ""
		if len(args) > 0 {
			app = args[0]
		}
		err := packager.Load(app)
		if err != nil {
			fmt.Printf("%v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	if internal.Experimental == "on" {
		rootCmd.AddCommand(loadCmd)
	}
}
