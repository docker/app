package cmd

import (
	"github.com/docker/lunchbox/packager"
	"github.com/spf13/cobra"
)

// inspectCmd represents the inspect command
var inspectCmd = &cobra.Command{
	Use:   "inspect <app-name>",
	Short: "Retrieve metadata for a given app package",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return packager.Inspect(firstOrEmpty(args))
	},
}

func init() {
	rootCmd.AddCommand(inspectCmd)
}
