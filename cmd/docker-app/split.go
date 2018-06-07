package main

import (
	"github.com/docker/app/internal"
	"github.com/docker/app/internal/packager"
	"github.com/docker/cli/cli"
	"github.com/spf13/cobra"
)

var splitOutputDir string

func splitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "split [<app-name>] [-o output_dir]",
		Short: "Split a single-file application into multiple files",
		Args:  cli.RequiresMaxArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return packager.Split(firstOrEmpty(args), splitOutputDir)
		},
	}
	if internal.Experimental == "on" {
		cmd.Flags().StringVarP(&splitOutputDir, "output", "o", "-", "Output directory")
	}
	return cmd
}
