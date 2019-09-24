package commands

import (
	"fmt"

	"github.com/docker/app/internal/packager"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/spf13/cobra"
)

var (
	initComposeFile string
)

func initCmd(dockerCli command.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "init APP_NAME [--compose-file COMPOSE_FILE] [OPTIONS]",
		Short:   "Initialize Docker Application definition",
		Long:    `Start building a Docker Application package. If there is a docker-compose.yml file in the current directory it will be copied and used.`,
		Example: `$ docker app init myapp`,
		Args:    cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			created, err := packager.Init(args[0], initComposeFile)
			if err != nil {
				return err
			}
			fmt.Fprintf(dockerCli.Out(), "Created %q\n", created)
			return nil
		},
	}
	cmd.Flags().StringVar(&initComposeFile, "compose-file", "", "Compose file to use as application base (optional)")
	return cmd
}
