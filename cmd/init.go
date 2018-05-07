package cmd

import (
	"github.com/docker/cli/cli"
	"github.com/docker/lunchbox/packager"
	"github.com/spf13/cobra"
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init <app-name> [-c <compose-file>]",
	Short: "Initialize an app package in the current working directory",
	Long: `This command initializes a new app package. If the -c option is used, or
if a file named docker-compose.yml is found in the current directory, this file
and the associated .env file if found will be used as the base for this application.`,
	Args: cli.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return packager.Init(args[0], composeFile)
	},
}

var composeFile string

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().StringVarP(&composeFile, "compose-file", "c", "", "Initial Compose file (optional)")
}
