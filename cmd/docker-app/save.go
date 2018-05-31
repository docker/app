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
			_, err := packager.Save(firstOrEmpty(args), savePrefix, saveTag)
			return err
		},
	}
	cmd.Flags().StringVarP(&savePrefix, "prefix", "p", "", "prefix to use (default: repository_prefix in metadata)")
	cmd.Flags().StringVarP(&saveTag, "tag", "t", "", "tag to use (default: version in metadata)")
	return cmd
}
