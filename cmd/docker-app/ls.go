package main

import (
	"github.com/docker/app/internal/image"
	"github.com/docker/cli/cli"
	"github.com/spf13/cobra"
)

type listOptions struct {
	quiet bool
}

func lsCmd() *cobra.Command {
	var opts listOptions
	cmd := &cobra.Command{
		Use:   "ls [<app-name>:[<tag>]]",
		Short: "List applications.",
		Args:  cli.RequiresMaxArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return image.List(firstOrEmpty(args), opts.quiet)
		},
	}
	cmd.Flags().BoolVarP(&opts.quiet, "quiet", "q", false, "Only show numeric IDs")
	return cmd
}
