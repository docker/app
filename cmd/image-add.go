package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/docker/lunchbox/image"
	"github.com/spf13/cobra"
)

var imageAddCmd = &cobra.Command{
	Use:   "image-add <app-name> [services...]",
	Short: "Add images for given services (default: all) to the app package",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		d := make(map[string]string)
		for _, v := range imageAddEnv {
			kv := strings.SplitN(v, "=", 2)
			if len(kv) != 2 {
				fmt.Printf("Malformed env input: '%s'\n", v)
				os.Exit(1)
			}
			d[kv[0]] = kv[1]
		}
		err := image.Add(args[0], args[1:], imageAddComposeFiles, imageAddSettingsFile, d)
		if err != nil {
			fmt.Printf("%v\n", err)
			os.Exit(1)
		}
	},
}
var imageAddComposeFiles []string
var imageAddSettingsFile []string
var imageAddEnv []string

func init() {
	rootCmd.AddCommand(imageAddCmd)
	imageAddCmd.Flags().StringArrayVarP(&imageAddComposeFiles, "compose-files", "c", []string{}, "Override Compose files")
	imageAddCmd.Flags().StringArrayVarP(&imageAddSettingsFile, "settings-files", "s", []string{}, "Override settings files")
	imageAddCmd.Flags().StringArrayVarP(&imageAddEnv, "env", "e", []string{}, "Override environment values")
}
