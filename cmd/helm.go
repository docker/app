package cmd

import (
	"github.com/docker/cli/cli"
	"github.com/docker/lunchbox/internal"
	"github.com/docker/lunchbox/renderer"
	"github.com/spf13/cobra"
)

var helmCmd = &cobra.Command{
	Use:   "helm [<app-name>] [-s key=value...] [-f settings-file...]",
	Short: "Render the Compose file for this app as an Helm package",
	Long: `The helm command creates or updates the directory <app-name>.chart.
- Chart.yaml is created or updated from the app's metadata.
- values.yaml is created or updated with the values from settings which are
  actually used by the compose file.
- templates/stack.yaml is created, with a stack template extracted from the app's
docker-compose.yml. If the --render option is used, the docker-compose.yml will
be rendered instead of exported as a template. Note that template export will
not work if you use go templating, only variable substitution is supported.`,
	Args: cli.RequiresMaxArgs(1),
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
	}
	helmCmd.Flags().StringArrayVarP(&helmSettingsFile, "settings-files", "f", []string{}, "Override settings files")
	helmCmd.Flags().StringArrayVarP(&helmEnv, "set", "s", []string{}, "Override environment values")
	helmCmd.Flags().BoolVarP(&helmRender, "render", "r", false, "Render the template instead of exporting it")
}
