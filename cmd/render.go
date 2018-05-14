package cmd

import (
	"fmt"
	"os"

	"github.com/docker/cli/cli"
	"github.com/docker/lunchbox/internal"
	"github.com/docker/lunchbox/renderer"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

var renderCmd = &cobra.Command{
	Use:   "render <app-name> [-s key=value...] [-f settings-file...]",
	Short: "Render the Compose file for this app",
	Long: `Render generates a Compose file from the application's template and optional additional files.
Override is provided in different ways:
- External YAML settings files can be specified with the -f flag. All settings
  files are merged in order, the app's settings coming first.
- Individual settings values can be passed directly on the command line with the
  -s flag. These value takes precedence over all settings files.
`,
	Args: cli.RequiresMaxArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		d, err := parseSettings(renderEnv)
		if err != nil {
			return err
		}
		rendered, err := renderer.Render(firstOrEmpty(args), renderComposeFiles, renderSettingsFile, d)
		if err != nil {
			return err
		}
		res, err := yaml.Marshal(rendered)
		if err != nil {
			return err
		}
		if renderOutput == "-" {
			fmt.Print(string(res))
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

var renderComposeFiles []string
var renderSettingsFile []string
var renderEnv []string
var renderOutput string

func init() {
	rootCmd.AddCommand(renderCmd)
	if internal.Experimental == "on" {
		renderCmd.Use += " [-c <compose-files>...]"
		renderCmd.Long += `- External Compose files or template Compose files can be specified with the -c flag.
  (Repeat the flag for multiple files). These files will be merged in order with
  the app's own Compose file.`
		renderCmd.Flags().StringArrayVarP(&renderComposeFiles, "compose-files", "c", []string{}, "Override Compose files")
	}
	renderCmd.Flags().StringArrayVarP(&renderSettingsFile, "settings-files", "f", []string{}, "Override settings files")
	renderCmd.Flags().StringArrayVarP(&renderEnv, "set", "s", []string{}, "Override settings values")
	renderCmd.Flags().StringVarP(&renderOutput, "output", "o", "-", "Output file")
}
