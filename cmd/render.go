package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/docker/lunchbox/renderer"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

var renderCmd = &cobra.Command{
	Use:   "render <app-name> [-c <compose-files>...] [-e key=value...] [-s settings-file...]",
	Short: "Render the Compose file for this app",
	Long: `Render generates a Compose file from the application's template and optional additional files.
Override is provided in three different ways:
- External Compose files or template Compose files can be specified with the -c flag.
  (Repeat the flag for multiple files). These files will be merged in order with
  the app's own Compose file.
- External YAML settings files can be specified with the -s flag. All settings
  files are merged in order, the app's settings coming first.
- Individual settings values can be passed directly on the command line with the
  -e flag. These value takes precedence over all settings files.
`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		d := make(map[string]string)
		for _, v := range renderEnv {
			kv := strings.SplitN(v, "=", 2)
			if len(kv) != 2 {
				fmt.Printf("Malformed env input: '%s'\n", v)
				os.Exit(1)
			}
			d[kv[0]] = kv[1]
		}
		rendered, err := renderer.Render(args[0], renderComposeFiles, renderSettingsFile, d)
		if err != nil {
			fmt.Printf("%v\n", err)
			os.Exit(1)
		}
		res, err := yaml.Marshal(rendered)
		if err != nil {
			fmt.Printf("%v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(res))
	},
}

var renderComposeFiles []string
var renderSettingsFile []string
var renderEnv []string

func init() {
	rootCmd.AddCommand(renderCmd)
	renderCmd.Flags().StringArrayVarP(&renderComposeFiles, "compose-files", "c", []string{}, "Override Compose files")
	renderCmd.Flags().StringArrayVarP(&renderSettingsFile, "settings-files", "s", []string{}, "Override settings files")
	renderCmd.Flags().StringArrayVarP(&renderEnv, "env", "e", []string{}, "Override settings values")
}
