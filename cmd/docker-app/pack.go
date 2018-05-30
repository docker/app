package main

import (
	"github.com/docker/cli/cli"
	"github.com/docker/lunchbox/packager"
	"github.com/spf13/cobra"
)

var packCmd = &cobra.Command{
	Use:   "pack [<app-name>] [-o output_file]",
	Short: "Pack the application as a single file",
	Args:  cli.RequiresMaxArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return packager.Pack(firstOrEmpty(args), packOutputFile)
	},
}

var packOutputFile string

func init() {
	rootCmd.AddCommand(packCmd)
	packCmd.Flags().StringVarP(&packOutputFile, "output", "o", "-", "Output file (- for stdout)")
}
