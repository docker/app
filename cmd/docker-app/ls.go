package main

import (
	"github.com/docker/app/internal/image"
	"github.com/docker/cli/cli"
	"github.com/spf13/cobra"
)

func lsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ls [<app-name>:[<tag>]]",
		Short: "List applications.",
		Args:  cli.RequiresMaxArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return image.List(firstOrEmpty(args))
		},
	}
	return cmd
}
