package main

import (
	"fmt"
	"strings"

	"github.com/docker/lunchbox/image"
	"github.com/docker/lunchbox/internal"
	"github.com/spf13/cobra"
)

var (
	imageAddComposeFiles []string
	imageAddSettingsFile []string
	imageAddEnv          []string
)

func imageAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "image-add <app-name> [services...]",
		Short: "Add images for given services (default: all) to the app package",
		Long: `This command renders the app's docker-compose.yml file, looks for the
images it uses, and saves them from the local docker daemon to the images/
subdirectory.`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			d := make(map[string]string)
			for _, v := range imageAddEnv {
				kv := strings.SplitN(v, "=", 2)
				if len(kv) != 2 {
					return fmt.Errorf("Malformed env input: '%s'", v)
				}
				d[kv[0]] = kv[1]
			}
			return image.Add(args[0], args[1:], imageAddComposeFiles, imageAddSettingsFile, d)
		},
	}
	if internal.Experimental == "on" {
		cmd.Flags().StringArrayVarP(&imageAddComposeFiles, "compose-files", "c", []string{}, "Override Compose files")
		cmd.Flags().StringArrayVarP(&imageAddSettingsFile, "settings-files", "s", []string{}, "Override settings files")
		cmd.Flags().StringArrayVarP(&imageAddEnv, "env", "e", []string{}, "Override environment values")
	}
	return cmd
}
