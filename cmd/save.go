package cmd

import (
	"github.com/docker/cli/cli"
	"github.com/docker/lunchbox/internal"
	"github.com/docker/lunchbox/packager"
	"github.com/spf13/cobra"
)

var saveCmd = &cobra.Command{
	Use:   "save <app-name>",
	Short: "Save the application to docker (in preparation for push)",
	Args:  cli.RequiresMaxArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if saveTag == "" {
			saveTag = "latest"
		}
		app := ""
		if len(args) > 0 {
			app = args[0]
		}
		return packager.Save(app, savePrefix, saveTag)
	},
}

var (
	savePrefix string
	saveTag    string
)

func init() {
	if internal.Experimental == "on" {
		rootCmd.AddCommand(saveCmd)
		saveCmd.Flags().StringVarP(&savePrefix, "prefix", "p", "", "prefix to use")
		saveCmd.Flags().StringVarP(&saveTag, "tag", "t", "latest", "tag to use")
	}
}
