package cmd

import (
	"fmt"
	"os"

	"github.com/docker/lunchbox/internal"
	"github.com/docker/lunchbox/packager"
	"github.com/spf13/cobra"
)

var pullCmd = &cobra.Command{
	Use:   "pull <repotag>",
	Short: "Pull an app from a registry",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		err := packager.Pull(args[0])
		if err != nil {
			fmt.Printf("%v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	if internal.Experimental == "on" {
		rootCmd.AddCommand(pullCmd)
	}
}
