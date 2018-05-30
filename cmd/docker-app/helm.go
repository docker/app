package main

import (
	"github.com/docker/cli/cli"
	"github.com/docker/lunchbox/internal"
	"github.com/docker/lunchbox/renderer"
	"github.com/spf13/cobra"
)

var (
	helmComposeFiles []string
	helmSettingsFile []string
	helmEnv          []string
	helmRender       bool
)

func helmCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "helm [<app-name>] [-s key=value...] [-f settings-file...]",
		Short: "Generate a Helm chart",
		Long:  `Generate a Helm chart for the application.`,
		Args:  cli.RequiresMaxArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := parseSettings(helmEnv)
			if err != nil {
				return err
			}
			return renderer.Helm(firstOrEmpty(args), helmComposeFiles, helmSettingsFile, d, helmRender)
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
	return cmd
}
