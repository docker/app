package cmd

import (
	"fmt"
	"os"

	"github.com/docker/lunchbox/internal"
	"github.com/docker/lunchbox/packager"
	"github.com/spf13/cobra"
)

var unpackCmd = &cobra.Command{
	Use:   "unpack <app-name> [-o output_dir]",
	Short: "Unpack the app to expose the content",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		err := packager.Unpack(args[0], unpackOutputDir)
		if err != nil {
			fmt.Printf("%v\n", err)
			os.Exit(1)
		}
	},
}

var unpackOutputDir string

func init() {
	if internal.Experimental == "on" {
		rootCmd.AddCommand(unpackCmd)
		unpackCmd.Flags().StringVarP(&unpackOutputDir, "output", "o", ".", "Output directory (.)")
	}
}
