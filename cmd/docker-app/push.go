package main

import (
	"github.com/docker/cli/cli"
	"github.com/docker/lunchbox/packager"
	"github.com/spf13/cobra"
)

var (
	pushPrefix string
	pushTag    string
)

func pushCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "push [<app-name>]",
		Short: "Push the application to a registry",
		Args:  cli.RequiresMaxArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return packager.Push(firstOrEmpty(args), pushPrefix, pushTag)
		},
	}
	cmd.Flags().StringVarP(&pushPrefix, "prefix", "p", "", "prefix to use (default: repository_prefix in metadata)")
	cmd.Flags().StringVarP(&pushTag, "tag", "t", "", "tag to use (default: version in metadata")
	return cmd
}
