package cmd

import (
	"fmt"
	"os"

	"github.com/docker/lunchbox/packager"
	"github.com/spf13/cobra"
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init <app-name> [-c <compose-files>...]",
	Short: "Initialize an app package in the current working directory",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("init called")
		if err := packager.Init(args[0], composeFiles); err != nil {
			fmt.Printf("%v\n", err)
			os.Exit(1)
		}
	},
}

var composeFiles []string

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().StringArrayVarP(&composeFiles, "compose-files", "c", []string{}, "Initial Compose files (optional)")
}
