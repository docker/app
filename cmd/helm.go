package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/docker/lunchbox/renderer"
	"github.com/spf13/cobra"
)

var helmCmd = &cobra.Command{
	Use:   "helm <app-name> [-c <compose-files>...] [-e key=value...] [-f settings-file...]",
	Short: "Render the composefile for this app as an Helm package",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		d := make(map[string]string)
		for _, v := range helmEnv {
			kv := strings.SplitN(v, "=", 2)
			if len(kv) != 2 {
				fmt.Printf("Malformed env input: '%s'\n", v)
				os.Exit(1)
			}
			d[kv[0]] = kv[1]
		}
		err := renderer.Helm(args[0], helmComposeFiles, helmSettingsFile, d)
		if err != nil {
			fmt.Printf("%v\n", err)
			os.Exit(1)
		}
	},
}

var helmComposeFiles []string
var helmSettingsFile []string
var helmEnv []string

func init() {
	rootCmd.AddCommand(helmCmd)
	helmCmd.Flags().StringArrayVarP(&helmComposeFiles, "compose-files", "c", []string{}, "Override Compose files")
	helmCmd.Flags().StringArrayVarP(&helmSettingsFile, "settings-files", "s", []string{}, "Override settings files")
	helmCmd.Flags().StringArrayVarP(&helmEnv, "env", "e", []string{}, "Override environment values")
}
