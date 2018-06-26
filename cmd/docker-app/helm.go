package main

import (
	"fmt"

	"github.com/docker/app/internal"
	"github.com/docker/app/internal/packager"
	"github.com/docker/app/internal/renderer"
	"github.com/docker/cli/cli"
	"github.com/spf13/cobra"
)

var (
	helmComposeFiles []string
	helmSettingsFile []string
	helmEnv          []string
	helmRender       bool
	stackVersion     string
)

func helmCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "helm [<app-name>] [-s key=value...] [-f settings-file...]",
		Short: "Generate a Helm chart",
		Long:  `Generate a Helm chart for the application.`,
		Args:  cli.RequiresMaxArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			appname, cleanup, err := packager.Extract(firstOrEmpty(args))
			if err != nil {
				return err
			}
			defer cleanup()
			d, err := parseSettings(helmEnv)
			if err != nil {
				return err
			}
			if stackVersion != renderer.V1Beta1 && stackVersion != renderer.V1Beta2 {
				return fmt.Errorf("invalid stack version %q (accepted values: %s, %s)", stackVersion, renderer.V1Beta1, renderer.V1Beta2)
			}
			return renderer.Helm(appname, helmComposeFiles, helmSettingsFile, d, helmRender, stackVersion)
		},
	}
	if internal.Experimental == "on" {
		cmd.Flags().StringArrayVarP(&helmComposeFiles, "compose-files", "c", []string{}, "Override Compose files")
		cmd.Use += " [-c <compose-files>...]"
		cmd.Flags().BoolVarP(&helmRender, "render", "r", false, "Render the template instead of exporting it")
		cmd.Long += ` If the --render option is used, the docker-compose.yml will
be rendered instead of exported as a template.`
	}
	cmd.Flags().StringArrayVarP(&helmSettingsFile, "settings-files", "f", []string{}, "Override settings files")
	cmd.Flags().StringArrayVarP(&helmEnv, "set", "s", []string{}, "Override settings values")
	cmd.Flags().StringVarP(&stackVersion, "stack-version", "", renderer.V1Beta2, "Version of the stack specification for the produced helm chart")
	return cmd
}
