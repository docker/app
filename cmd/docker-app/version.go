package main

import (
	"fmt"

	"github.com/docker/lunchbox/internal"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(internal.FullVersion())
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
