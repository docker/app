package main

import (
	"github.com/docker/cli/cli"
	"github.com/docker/lunchbox/internal"
	"github.com/docker/lunchbox/renderer"
	"github.com/spf13/cobra"
)

var helmCmd = &cobra.Command{
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

var helmComposeFiles []string
var helmSettingsFile []string
var helmEnv []string
var helmRender bool

func init() {
	rootCmd.AddCommand(helmCmd)
	if internal.Experimental == "on" {
		helmCmd.Flags().StringArrayVarP(&helmComposeFiles, "compose-files", "c", []string{}, "Override Compose files")
		helmCmd.Use += " [-c <compose-files>...]"
		helmCmd.Flags().BoolVarP(&helmRender, "render", "r", false, "Render the template instead of exporting it")
		helmCmd.Long += ` If the --render option is used, the docker-compose.yml will
be rendered instead of exported as a template.`
	}
	helmCmd.Flags().StringArrayVarP(&helmSettingsFile, "settings-files", "f", []string{}, "Override settings files")
	helmCmd.Flags().StringArrayVarP(&helmEnv, "set", "s", []string{}, "Override settings values")
}
