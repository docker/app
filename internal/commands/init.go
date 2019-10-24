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
		Use:   "init [OPTIONS] APP_DEFINITION",
		Short: "Initialize an App definition",
		Example: `$ docker app init myapp
$ docker app init myapp --compose-file docker-compose.yml`,
		Args: cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			created, err := packager.Init(args[0], initComposeFile)
			if err != nil {
				return err
			}
			fmt.Fprintf(dockerCli.Out(), "Created %q\n", created)
			return nil
		},
	}
	cmd.Flags().StringVar(&initComposeFile, "compose-file", "", "Compose file to use to bootstrap a Docker App definition")
	return cmd
}
