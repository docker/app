package cmd

import (
	"fmt"
	"os"

	"github.com/docker/lunchbox/internal"
	"github.com/docker/lunchbox/packager"
	"github.com/spf13/cobra"
)

var pushCmd = &cobra.Command{
	Use:   "push <app-name>",
	Short: "Push the application to a registry",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if pushTag == "" {
			pushTag = "latest"
		}
		app := ""
		if len(args) > 0 {
			app = args[0]
		}
		err := packager.Push(app, pushPrefix, pushTag)
		if err != nil {
			fmt.Printf("%v\n", err)
			os.Exit(1)
		}
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
