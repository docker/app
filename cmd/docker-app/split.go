package main

import (
	"github.com/docker/cli/cli"
	"github.com/docker/lunchbox/internal"
	"github.com/docker/lunchbox/packager"
	"github.com/spf13/cobra"
)

var splitCmd = &cobra.Command{
	Use:   "split [<app-name>] [-o output_dir]",
	Short: "Split a single-file application into multiple files",
	Args:  cli.RequiresMaxArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return packager.Split(firstOrEmpty(args), splitOutputDir)
	},
}

var splitOutputDir string

func init() {
	if internal.Experimental == "on" {
		rootCmd.AddCommand(splitCmd)
		splitCmd.Flags().StringVarP(&splitOutputDir, "output", "o", "-", "Output directory")
	}
}
