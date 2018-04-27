package cmd

import (
	"github.com/docker/cli/cli"
	"github.com/docker/lunchbox/packager"
	"github.com/spf13/cobra"
)

// inspectCmd represents the inspect command
var inspectCmd = &cobra.Command{
	Use:   "inspect <app-name>",
	Short: "Retrieve metadata for a given app package",
	Args:  cli.RequiresMaxArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		app := ""
		if len(args) > 0 {
			app = args[0]
		}
		return packager.Inspect(app)
	},
}

func init() {
	rootCmd.AddCommand(inspectCmd)
}
