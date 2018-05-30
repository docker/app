package main

import (
	"github.com/docker/cli/cli"
	"github.com/docker/lunchbox/internal"
	"github.com/docker/lunchbox/packager"
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
