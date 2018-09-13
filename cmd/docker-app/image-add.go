package main

import (
	"os"

	"github.com/docker/app/internal"
	"github.com/docker/app/internal/image"
	"github.com/docker/app/internal/packager"
	"github.com/docker/app/render"
	"github.com/docker/app/types"
	cliopts "github.com/docker/cli/opts"
	"github.com/spf13/cobra"
)

var (
	imageAddComposeFiles []string
	imageAddSettingsFile []string
	imageAddEnv          []string
	imageAddPull         bool
	imageAddQuiet        bool
)

func imageAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add <app-name> [--pull] [services...]",
		Short: "Add images for given services (default: all) to the app package",
		Long: `This command renders the app's docker-compose.yml file, looks for the
images it uses, and saves them from the local docker daemon to the images/
subdirectory.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			var (
				oappname string
				err      error
				services []string
			)
			if len(args) == 0 {
				oappname, err = packager.FindApp()
				if err != nil {
					return err
				}
			} else {
				oappname = args[0]
				services = args[1:]
			}
			app, err := packager.Extract(oappname,
				types.WithSettingsFiles(imageAddSettingsFile...),
				types.WithComposeFiles(imageAddComposeFiles...),
			)
			if err != nil {
				return err
			}
			defer app.Cleanup()
			d := cliopts.ConvertKVStringsToMap(imageAddEnv)
			config, err := render.Render(app, d)
			if err != nil {
				return err
			}
			if err := image.Add(app.Path, services, config, imageAddPull, imageAddQuiet); err != nil {
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
				return packager.Pack(app.Name, target)
			}
			return nil
		},
	}
	if internal.Experimental == "on" {
		cmd.Flags().StringArrayVarP(&imageAddComposeFiles, "compose-files", "c", []string{}, "Override Compose files")
	}
	cmd.Flags().StringArrayVarP(&imageAddSettingsFile, "settings-files", "f", []string{}, "Override settings files")
	cmd.Flags().StringArrayVarP(&imageAddEnv, "set", "s", []string{}, "Override environment values")
	cmd.Flags().BoolVarP(&imageAddPull, "pull", "p", false, "Pull images first")
	cmd.Flags().BoolVarP(&imageAddPull, "quiet", "q", false, "Suppress progress output")

	return cmd
}
