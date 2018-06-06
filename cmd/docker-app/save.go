package main

import (
	"fmt"

	"github.com/docker/app/packager"
	"github.com/docker/cli/cli"
	"github.com/spf13/cobra"
)

var (
	savePrefix string
	saveTag    string
)

func saveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "save [<app-name>]",
		Short: "Save the application as an image to the docker daemon(in preparation for push)",
		Args:  cli.RequiresMaxArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			imageName, err := packager.Save(firstOrEmpty(args), savePrefix, saveTag)
			if imageName != "" {
				fmt.Printf("Saved application as image: %s\n", imageName)
			}
			return err
		},
	}
	cmd.Flags().StringVarP(&savePrefix, "prefix", "p", "", "prefix to use (default: repository_prefix in metadata)")
	cmd.Flags().StringVarP(&saveTag, "tag", "t", "", "tag to use (default: version in metadata)")
	return cmd
}
