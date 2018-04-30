package cmd

import (
	"github.com/docker/lunchbox/internal"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// buildCmd represents the build command
var buildCmd = &cobra.Command{
	Use:   "build <app-name>",
	Short: "Compile an app package from locally available data",
	Run: func(cmd *cobra.Command, args []string) {
		log.Info("build called")
	},
}

func init() {
	if internal.Experimental == "on" {
		rootCmd.AddCommand(buildCmd)
	}
}
