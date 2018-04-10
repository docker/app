package cmd

import (
	"fmt"
	"os"

	"github.com/docker/lunchbox/packager"
	"github.com/spf13/cobra"
)

var loadCmd = &cobra.Command{
	Use:   "load <repotag>",
	Short: "Load an app from docker",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		err := packager.Load(args[0])
		if err != nil {
			fmt.Printf("%v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(loadCmd)
}
