package main

import (
	"github.com/docker/app/packager"
	"github.com/docker/cli/cli"
	"github.com/spf13/cobra"
)

var packOutputFile string

func packCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pack [<app-name>] [-o output_file]",
		Short: "Pack the application as a single file",
		Args:  cli.RequiresMaxArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return packager.Pack(firstOrEmpty(args), packOutputFile)
		},
	}
	cmd.Flags().StringVarP(&packOutputFile, "output", "o", "-", "Output file (- for stdout)")
	return cmd
}
