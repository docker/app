package cmd

import (
	"github.com/docker/cli/cli"
	"github.com/docker/lunchbox/packager"
	"github.com/spf13/cobra"
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init <app-name> [-c <compose-file>] [-d <description>] [-m name:email ...]",
	Short: "Start building a Docker application",
	Long:  `Start building a Docker application. Will automatically detect a docker-compose.yml file in the current directory.`,
	Args:  cli.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return packager.Init(args[0], initComposeFile, initDescription, initMaintainers, initSingleFile)
	},
}

var initComposeFile string
var initDescription string
var initMaintainers []string
var initSingleFile bool

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().StringVarP(&initComposeFile, "compose-file", "c", "", "Initial Compose file (optional)")
	initCmd.Flags().StringVarP(&initDescription, "description", "d", "", "Initial description (optional)")
	initCmd.Flags().StringArrayVarP(&initMaintainers, "maintainer", "m", []string{}, "Maintainer (name:email) (optional)")
	initCmd.Flags().BoolVarP(&initSingleFile, "single-file", "s", false, "Create a single-file application")
}
