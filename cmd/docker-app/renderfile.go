package main

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/docker/app/internal/packager"
	"github.com/docker/app/render"
	"github.com/docker/app/types"
	"github.com/docker/cli/cli"
	cliopts "github.com/docker/cli/opts"
	"github.com/spf13/cobra"
)

type renderFileOptions struct {
	renderSettingsFile []string
	renderEnv          []string
}

func renderFileCmd() *cobra.Command {
	var opts renderFileOptions
	cmd := &cobra.Command{
		Use:   "render-attachment [<app-path>] <file-path>",
		Short: "render specified file",
		Args:  cli.RequiresRangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			var appname, filename string
			if len(args) == 1 {
				filename = args[0]
			} else {
				appname = args[0]
				filename = args[1]
			}
			app, err := packager.Extract(appname,
				types.WithSettingsFiles(opts.renderSettingsFile...))
			if err != nil {
				return err
			}
			data, err := ioutil.ReadFile(filepath.Join(app.Path, filename))
			if err != nil {
				return err
			}
			d := cliopts.ConvertKVStringsToMap(opts.renderEnv)
			result, err := render.RenderConfig(app, d, string(data))
			if err != nil {
				return err
			}
			fmt.Printf("%s", result)
			return nil
		},
	}
	cmd.Flags().StringArrayVarP(&opts.renderSettingsFile, "settings-files", "f", []string{}, "Override settings files")
	cmd.Flags().StringArrayVarP(&opts.renderEnv, "set", "s", []string{}, "Override settings values")
	return cmd
}
