package cmd

import (
	"github.com/docker/cli/cli"
	"github.com/docker/lunchbox/internal"
	"github.com/docker/lunchbox/packager"
	"github.com/spf13/cobra"
)

var unpackCmd = &cobra.Command{
	Use:   "unpack <app-name> [-o output_dir]",
	Short: "Unpack the app to expose the content",
	Args:  cli.RequiresMaxArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		app := ""
		if len(args) > 0 {
			app = args[0]
		}
		return packager.Unpack(app, unpackOutputDir)
	},
}

var unpackOutputDir string

func init() {
	if internal.Experimental == "on" {
		rootCmd.AddCommand(unpackCmd)
		unpackCmd.Flags().StringVarP(&unpackOutputDir, "output", "o", ".", "Output directory (.)")
	}
}
