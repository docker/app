package main

import (
	"github.com/docker/cli/cli"
	"github.com/docker/lunchbox/packager"
	"github.com/spf13/cobra"
)

var unpackCmd = &cobra.Command{
	Use:   "unpack <app-name> [-o output_dir]",
	Short: "Unpack the application to expose the content",
	Args:  cli.RequiresMaxArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return packager.Unpack(firstOrEmpty(args), unpackOutputDir)
	},
}

var unpackOutputDir string

func init() {
	rootCmd.AddCommand(unpackCmd)
	unpackCmd.Flags().StringVarP(&unpackOutputDir, "output", "o", ".", "Output directory (.)")
}
