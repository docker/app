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
		Use:     "init APP_NAME [--compose-file COMPOSE_FILE] [--description DESCRIPTION] [--maintainer NAME:EMAIL ...] [OPTIONS]",
		Short:   "Initialize Docker Application definition",
		Long:    `Start building a Docker Application package. If there is a docker-compose.yml file in the current directory it will be copied and used.`,
		Example: `$ docker app init myapp --description "a useful description"`,
		Args:    cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return packager.Init(args[0], initComposeFile, initDescription, initMaintainers, initSingleFile)
		},
	}
	cmd.Flags().StringVar(&initComposeFile, "compose-file", "", "Compose file to use as application base (optional)")
	cmd.Flags().StringVar(&initDescription, "description", "", "Human readable description of your application (optional)")
	cmd.Flags().StringArrayVar(&initMaintainers, "maintainer", []string{}, "Name and email address of person responsible for the application (name:email) (optional)")
	cmd.Flags().BoolVar(&initSingleFile, "single-file", false, "Create a single-file Docker Application definition")
	return cmd
}
