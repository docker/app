package main

import (
	"github.com/docker/cli/cli"
	"github.com/docker/lunchbox/packager"
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
			if saveTag == "" {
				saveTag = "latest"
			}
			return packager.Save(firstOrEmpty(args), savePrefix, saveTag)
		},
	}
	cmd.Flags().StringVarP(&savePrefix, "prefix", "p", "", "prefix to use")
	cmd.Flags().StringVarP(&saveTag, "tag", "t", "latest", "tag to use")
	return cmd
}
