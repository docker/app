package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/docker/lunchbox/internal"
	"github.com/docker/lunchbox/renderer"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

var renderCmd = &cobra.Command{
	Use:   "render <app-name> [-e key=value...] [-s settings-file...]",
	Short: "Render the Compose file for this app",
	Long: `Render generates a Compose file from the application's template and optional additional files.
Override is provided in different ways:
- External YAML settings files can be specified with the -f flag. All settings
  files are merged in order, the app's settings coming first.
- Individual settings values can be passed directly on the command line with the
  -s flag. These value takes precedence over all settings files.
`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		d := make(map[string]string)
		for _, v := range renderEnv {
			kv := strings.SplitN(v, "=", 2)
			if len(kv) != 2 {
				fmt.Printf("Missing '=' in setting '%s', expected KEY=VALUE\n", v)
				os.Exit(1)
			}
			if _, ok := d[kv[0]]; ok {
				fmt.Printf("Duplicate command line setting: '%s'\n", kv[0])
				os.Exit(1)
			}
			d[kv[0]] = kv[1]
		}
		app := ""
		if len(args) > 0 {
			app = args[0]
		}
		rendered, err := renderer.Render(app, renderComposeFiles, renderSettingsFile, d)
		if err != nil {
			return err
		}
		res, err := yaml.Marshal(rendered)
		if err != nil {
			return err
		}
		fmt.Printf("%s", string(res))
		return nil
	},
}

var renderComposeFiles []string
var renderSettingsFile []string
var renderEnv []string

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
}
