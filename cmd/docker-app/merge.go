package main

import (
	"github.com/docker/cli/cli"
	"github.com/docker/lunchbox/internal"
	"github.com/docker/lunchbox/packager"
	"github.com/spf13/cobra"
)

var mergeOutputFile string

func mergeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "merge [<app-name>] [-o output_dir]",
		Short: "Merge the application as a single file multi-document YAML",
		Args:  cli.RequiresMaxArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return packager.Merge(firstOrEmpty(args), mergeOutputFile)
		},
	}
	if internal.Experimental == "on" {
		cmd.Flags().StringVarP(&mergeOutputFile, "output", "o", "-", "Output file (default: stdout)")
	}
	return cmd
}
