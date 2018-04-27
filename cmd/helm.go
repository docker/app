package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/docker/cli/cli"
	"github.com/docker/lunchbox/internal"
	"github.com/docker/lunchbox/renderer"
	"github.com/spf13/cobra"
)

var helmCmd = &cobra.Command{
	Use:   "helm <app-name> [-c <compose-files>...] [-e key=value...] [-f settings-file...]",
	Short: "Render the Compose file for this app as an Helm package",
	Args:  cli.RequiresMaxArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		d := make(map[string]string)
		for _, v := range helmEnv {
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
		return renderer.Helm(firstOrEmpty(args), helmComposeFiles, helmSettingsFile, d)
	},
}

var helmComposeFiles []string
var helmSettingsFile []string
var helmEnv []string

func init() {
	rootCmd.AddCommand(helmCmd)
	if internal.Experimental == "on" {
		helmCmd.Flags().StringArrayVarP(&helmComposeFiles, "compose-files", "c", []string{}, "Override Compose files")
	}
	helmCmd.Flags().StringArrayVarP(&helmSettingsFile, "settings-files", "f", []string{}, "Override settings files")
	helmCmd.Flags().StringArrayVarP(&helmEnv, "set", "s", []string{}, "Override environment values")
}
