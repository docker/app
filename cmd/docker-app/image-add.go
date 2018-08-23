package main

import (
	"os"

	"github.com/docker/app/internal"
	"github.com/docker/app/internal/image"
	"github.com/docker/app/internal/packager"
	"github.com/docker/app/internal/renderer"
	"github.com/spf13/cobra"
)

var (
	imageAddComposeFiles []string
	imageAddSettingsFile []string
	imageAddEnv          []string
)

func imageAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "image-add <app-name> [services...]",
		Short: "Add images for given services (default: all) to the app package",
		Long: `This command renders the app's docker-compose.yml file, looks for the
images it uses, and saves them from the local docker daemon to the images/
subdirectory.`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			oappname := args[0]
			appname, cleanup, err := packager.Extract(oappname, nil)
			if err != nil {
				return err
			}
			defer cleanup()

			d, err := parseSettings(imageAddEnv)
			if err != nil {
				return err
			}
			config, err := renderer.Render(appname, imageAddComposeFiles, imageAddSettingsFile, d)
			if err != nil {
				return err
			}
			if err := image.Add(appname, args[1:], config); err != nil {
				return err
			}
			// check if source was a tarball
			s, err := os.Stat(oappname)
			if err != nil {
				// try appending our extension
				oappname = internal.DirNameFromAppName(oappname)
				s, err = os.Stat(oappname)
				if err != nil {
					return err
				}
			}
			if !s.IsDir() {
				target, err := os.Create(oappname)
				if err != nil {
					return err
				}
				// source was a tarball, rebuild it
				return packager.Pack(appname, target)
			}
			return nil
		},
	}
	if internal.Experimental == "on" {
		cmd.Flags().StringArrayVarP(&imageAddComposeFiles, "compose-files", "c", []string{}, "Override Compose files")
		cmd.Flags().StringArrayVarP(&imageAddSettingsFile, "settings-files", "s", []string{}, "Override settings files")
		cmd.Flags().StringArrayVarP(&imageAddEnv, "env", "e", []string{}, "Override environment values")
	}
	return cmd
}
