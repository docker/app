package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/docker/app/internal/com"

	"github.com/docker/app/internal"
	"github.com/docker/app/internal/packager"
	"github.com/docker/app/internal/renderer"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

var (
	renderComposeFiles []string
	renderSettingsFile []string
	renderEnv          []string
	renderOutput       string
)

func renderCmd(dockerCli command.Cli, fs com.FrontServiceClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "render <app-name> [-s key=value...] [-f settings-file...]",
		Short: "Render the Compose file for the application",
		Long:  `Render the Compose file for the application.`,
		Args:  cli.RequiresMaxArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			appname, cleanup, err := packager.Extract(firstOrEmpty(args), fs)
			if err != nil {
				return err
			}
			defer cleanup()
			versionBytes, err := ioutil.ReadFile(filepath.Join(appname, internal.ToolchainVersionFile))
			if os.IsNotExist(err) {
				// as if version match
				versionBytes = []byte(internal.Version)
			} else if err != nil {
				return err
			}
			if string(versionBytes) != internal.Version {
				return &com.VersionMismatch{
					BackendVersion: internal.Version,
					PackageVersion: string(versionBytes),
				}
			}
			d, err := parseSettings(renderEnv)
			if err != nil {
				return err
			}
			rendered, err := renderer.Render(appname, renderComposeFiles, renderSettingsFile, d)
			if err != nil {
				return err
			}
			res, err := yaml.Marshal(rendered)
			if err != nil {
				return err
			}
			if renderOutput == "-" {
				fmt.Fprint(dockerCli.Out(), string(res))
			} else {
				f, err := os.Create(renderOutput)
				if err != nil {
					return err
				}
				fmt.Fprint(f, string(res))
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
	return cmd
}
