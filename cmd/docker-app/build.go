package main

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// buildCmd represents the build command
func buildCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "build <app-name>",
		Short: "Compile an app package from locally available data",
		Run: func(cmd *cobra.Command, args []string) {
			log.Info("build called")
		},
	}
}
