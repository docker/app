package cmd

import (
	"github.com/docker/lunchbox/internal"
	"github.com/docker/lunchbox/packager"
	"github.com/spf13/cobra"
)

var packCmd = &cobra.Command{
	Use:   "pack <app-name> [-o output_file]",
	Short: "Pack this app as a single file",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return packager.Pack(firstOrEmpty(args), packOutputFile)
	},
}

var packOutputFile string

func init() {
	if internal.Experimental == "on" {
		rootCmd.AddCommand(packCmd)
		packCmd.Flags().StringVarP(&packOutputFile, "output", "o", "-", "Output file (- for stdout)")
	}
}
