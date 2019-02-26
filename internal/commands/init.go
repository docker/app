package commands

import (
	"github.com/docker/app/internal/packager"
	"github.com/docker/cli/cli"
	"github.com/spf13/cobra"
)

var (
	initComposeFile string
	initDescription string
	initMaintainers []string
	initSingleFile  bool
)

func initCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init <app-name> [-c <compose-file>] [-d <description>] [-m name:email ...]",
		Short: "Start building a Docker application",
		Long:  `Start building a Docker application. Will automatically detect a docker-compose.yml file in the current directory.`,
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return packager.Init(args[0], initComposeFile, initDescription, initMaintainers, initSingleFile)
		},
	}
	cmd.Flags().StringVarP(&initComposeFile, "compose-file", "c", "", "Initial Compose file (optional)")
	cmd.Flags().StringVarP(&initDescription, "description", "d", "", "Initial description (optional)")
	cmd.Flags().StringArrayVarP(&initMaintainers, "maintainer", "m", []string{}, "Maintainer (name:email) (optional)")
	cmd.Flags().BoolVarP(&initSingleFile, "single-file", "s", false, "Create a single-file application")
	return cmd
}
