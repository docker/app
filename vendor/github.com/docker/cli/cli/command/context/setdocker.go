package context

import (
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/context/docker"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func newSetDockerEndpointCommand(dockerCli command.Cli) *cobra.Command {
	opts := &dockerEndpointOptions{}
	cmd := &cobra.Command{
		Use:   "set-docker-endpoint <context> [options]",
		Short: "Reset the docker endpoint of a context",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			endpoint, err := opts.toEndpoint(dockerCli, name)
			if err != nil {
				return errors.Wrap(err, "unable to create docker endpoint config")
			}
			return docker.Save(dockerCli.ContextStore(), endpoint)
		},
	}

	opts.addFlags(cmd.Flags(), "")
	return cmd
}
