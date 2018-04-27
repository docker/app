package cmd

import (
	"github.com/docker/lunchbox/internal"
	"github.com/docker/lunchbox/packager"
	"github.com/spf13/cobra"
)

var unpackCmd = &cobra.Command{
	Use:   "unpack <app-name> [-o output_dir]",
	Short: "Unpack the app to expose the content",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return packager.Unpack(firstOrEmpty(args), unpackOutputDir)
	},
}

var unpackOutputDir string

func init() {
	if internal.Experimental == "on" {
		rootCmd.AddCommand(unpackCmd)
		unpackCmd.Flags().StringVarP(&unpackOutputDir, "output", "o", ".", "Output directory (.)")
	}
}
