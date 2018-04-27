package cmd

import (
	"github.com/docker/lunchbox/image"
	"github.com/docker/lunchbox/internal"
	"github.com/spf13/cobra"
)

var imageLoadCmd = &cobra.Command{
	Use:   "image-load <app-name> [services...]",
	Short: "Load stored images for given services (default: all) to the local docker daemon",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return image.Load(args[0], args[1:])
	},
}

func init() {
	if internal.Experimental == "on" {
		rootCmd.AddCommand(imageLoadCmd)
	}
}
