package main

import (
	"fmt"
	"os"

	"github.com/docker/app/internal"
	"github.com/docker/app/internal/formatter"
	"github.com/docker/app/internal/image"
	"github.com/docker/app/internal/packager"
	"github.com/docker/app/render"
	"github.com/docker/app/types"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	cliopts "github.com/docker/cli/opts"
	"github.com/spf13/cobra"
)

var (
	formatDriver       string
	renderComposeFiles []string
	renderSettingsFile []string
	renderEnv          []string
	renderOutput       string
	renderRegistry     string
)

func renderCmd(dockerCli command.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "render <app-name> [-s key=value...] [-f settings-file...]",
		Short: "Render the Compose file for the application",
		Long:  `Render the Compose file for the application.`,
		Args:  cli.RequiresMaxArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := packager.Extract(firstOrEmpty(args),
				types.WithSettingsFiles(renderSettingsFile...),
				types.WithComposeFiles(renderComposeFiles...),
			)
			if err != nil {
				return err
			}
			defer app.Cleanup()
			d := cliopts.ConvertKVStringsToMap(renderEnv)
			rendered, err := render.Render(app, d)
			if err != nil {
				return err
			}
			if renderRegistry != "" {
				if err = image.ChangeAllImages(rendered, renderRegistry); err != nil {
					return err
				}
			}
			res, err := formatter.Format(rendered, formatDriver)
			if err != nil {
				return err
			}
			if renderOutput == "-" {
				fmt.Fprint(dockerCli.Out(), res)
			} else {
				f, err := os.Create(renderOutput)
				if err != nil {
					return err
				}
				fmt.Fprint(f, res)
			}
			return nil
		},
	}
	if internal.Experimental == "on" {
		cmd.Use += " [-c <compose-files>...]"
		cmd.Long += `- External Compose files or template Compose files can be specified with the -c flag.
  (Repeat the flag for multiple files). These files will be merged in order with
  the app's own Compose file.`
		cmd.Flags().StringArrayVarP(&renderComposeFiles, "compose-files", "c", []string{}, "Override Compose files")
	}
	cmd.Flags().StringArrayVarP(&renderSettingsFile, "settings-files", "f", []string{}, "Override settings files")
	cmd.Flags().StringArrayVarP(&renderEnv, "set", "s", []string{}, "Override settings values")
	cmd.Flags().StringVarP(&renderOutput, "output", "o", "-", "Output file")
	cmd.Flags().StringVar(&formatDriver, "formatter", "yaml", "Configure the output format (yaml|json)")
	cmd.Flags().StringVarP(&renderRegistry, "registry", "r", "", "Override registry for all images")
	return cmd
}
