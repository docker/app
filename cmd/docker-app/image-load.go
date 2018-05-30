package main

import (
	"github.com/docker/lunchbox/image"
	"github.com/spf13/cobra"
)

func imageLoadCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "image-load <app-name> [services...]",
		Short: "Load stored images for given services (default: all) to the local docker daemon",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return image.Load(args[0], args[1:])
		},
	}
}
