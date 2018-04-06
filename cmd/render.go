package cmd

import (
	"fmt"
	"github.com/docker/lunchbox/packager"
	"github.com/spf13/cobra"
	"os"
	"strings"
)

var renderCmd = &cobra.Command{
	Use:   "render <app-name> [-c <compose-files>...] [-e key=value...] [-f settings-file...]",
	Short: "Render the composefile for this app",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		d := make(map[string]string)
		for _, v := range(renderEnv) {
			kv := strings.SplitN(v, "=", 2)
			if len(kv) != 2 {
				fmt.Printf("Malformed env input: '%s'\n", v)
				os.Exit(1)
			}
			d[kv[0]] = kv[1]
		}
		res, err := packager.Render(args[0], renderComposeFiles, renderSettingsFile, d)
		if err != nil {
			fmt.Printf("%v\n", err)
			os.Exit(1)
		}
		fmt.Println(res)
	},
}

var renderComposeFiles []string
var renderSettingsFile []string
var renderEnv []string

func init() {
	rootCmd.AddCommand(renderCmd)
	renderCmd.Flags().StringArrayVarP(&renderComposeFiles, "compose-files", "c", []string{}, "Override Compose files")
	renderCmd.Flags().StringArrayVarP(&renderSettingsFile, "settings-files", "s", []string{}, "Override settings files")
	renderCmd.Flags().StringArrayVarP(&renderEnv, "env", "e", []string{}, "Override environment values")
}