package main

import (
	"github.com/docker/app/internal"
	"github.com/docker/app/internal/image"
	"github.com/docker/app/internal/packager"
	"github.com/docker/app/render"
	"github.com/docker/app/types"
	cliopts "github.com/docker/cli/opts"
	"github.com/spf13/cobra"
)

var (
	imagePushComposeFiles []string
	imagePushSettingsFile []string
	imagePushEnv          []string
	imagePushRegistry     string
)

func imagePushCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "push <app-name> <registry> [services...]",
		Short: "Push images for given services (default: all) to given registry",
		Long: `This command renders the app's docker-compose.yml file, and pushes
the images saved in the application to the given registry.`,
		Args: cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := packager.Extract(args[0],
				types.WithSettingsFiles(imagePushSettingsFile...),
				types.WithComposeFiles(imagePushComposeFiles...),
			)
			if err != nil {
				return err
			}
			defer app.Cleanup()
			d := cliopts.ConvertKVStringsToMap(imagePushEnv)
			config, err := render.Render(app, d)
			if err != nil {
				return err
			}
			return image.Push(app.Path, args[1], args[2:], config)
		},
	}
	if internal.Experimental == "on" {
		cmd.Flags().StringArrayVarP(&imagePushComposeFiles, "compose-files", "c", []string{}, "Override Compose files")
	}
	cmd.Flags().StringArrayVarP(&imagePushSettingsFile, "settings-files", "f", []string{}, "Override settings files")
	cmd.Flags().StringArrayVarP(&imagePushEnv, "set", "s", []string{}, "Override environment values")
	cmd.Flags().StringVarP(&imagePushRegistry, "registry", "r", "", "Registry to push to")
	return cmd
}
