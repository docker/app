package main

import (
	"github.com/docker/app/internal/image"
	"github.com/docker/app/internal/packager"
	"github.com/spf13/cobra"
)

func imageLoadCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "load <app-name> [services...]",
		Short: "Load stored images for given services (default: all) to the local docker daemon",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := packager.Extract(args[0])
			if err != nil {
				return err
			}
			defer app.Cleanup()
			return image.Load(app.Path, args[1:])
		},
	}
}
