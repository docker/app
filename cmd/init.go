package cmd

import (
	"fmt"
	"os"

	"github.com/docker/lunchbox/packager"
	"github.com/spf13/cobra"
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init <app-name> [-c <compose-file>]",
	Short: "Initialize an app package in the current working directory",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("init called")
		if err := packager.Init(args[0], composeFile); err != nil {
			fmt.Printf("%v\n", err)
			os.Exit(1)
		}
	},
}

var composeFile string

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().StringVarP(&composeFile, "compose-file", "c", "", "Initial Compose file (optional)")
}
