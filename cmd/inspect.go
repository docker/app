package cmd

import (
	"fmt"
	"os"

	"github.com/docker/lunchbox/packager"
	"github.com/spf13/cobra"
)

// inspectCmd represents the inspect command
var inspectCmd = &cobra.Command{
	Use:   "inspect <app-name>",
	Short: "Retrieve metadata for a given app package",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		err := packager.Inspect(args[0])
		if err != nil {
			fmt.Printf("%v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(inspectCmd)
}
