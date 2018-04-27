package cmd

import (
	"github.com/docker/cli/cli"
	"github.com/docker/lunchbox/internal"
	"github.com/docker/lunchbox/packager"
	"github.com/spf13/cobra"
)

var pushCmd = &cobra.Command{
	Use:   "push <app-name>",
	Short: "Push the application to a registry",
	Args:  cli.RequiresMaxArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if pushTag == "" {
			pushTag = "latest"
		}
		app := ""
		if len(args) > 0 {
			app = args[0]
		}
		return packager.Push(app, pushPrefix, pushTag)
	},
}

var (
	pushPrefix string
	pushTag    string
)

func init() {
	if internal.Experimental == "on" {
		rootCmd.AddCommand(pushCmd)
		pushCmd.Flags().StringVarP(&pushPrefix, "prefix", "p", "", "prefix to use")
		pushCmd.Flags().StringVarP(&pushTag, "tag", "t", "latest", "tag to use")
	}
}
