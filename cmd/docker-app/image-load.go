package main

import (
	"github.com/docker/app/internal/image"
	"github.com/docker/app/internal/packager"
	"github.com/spf13/cobra"
)

func imageLoadCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "image-load <app-name> [services...]",
		Short: "Load stored images for given services (default: all) to the local docker daemon",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			appname, cleanup, err := packager.Extract(args[0])
			if err != nil {
				return err
			}
			defer cleanup()
			return image.Load(appname, args[1:])
		},
	}
}
