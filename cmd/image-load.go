package cmd

import (
	"fmt"
	"os"

	"github.com/docker/lunchbox/image"
	"github.com/docker/lunchbox/internal"
	"github.com/spf13/cobra"
)

var imageLoadCmd = &cobra.Command{
	Use:   "image-load <app-name> [services...]",
	Short: "Load stored images for given services (default: all) to the local docker daemon",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		err := image.Load(args[0], args[1:])
		if err != nil {
			fmt.Printf("%v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	if internal.Experimental == "on" {
		rootCmd.AddCommand(imageLoadCmd)
	}
}
