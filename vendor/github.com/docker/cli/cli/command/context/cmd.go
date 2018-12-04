package context

import (
	"github.com/spf13/cobra"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
)

// NewContextCommand returns the context cli subcommand
func NewContextCommand(dockerCli command.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "context",
		Short: "Manage contexts",
		Args:  cli.NoArgs,
		RunE:  command.ShowHelp(dockerCli.Err()),
	}
	cmd.AddCommand(
		newCreateCommand(dockerCli),
		newListCommand(dockerCli),
		newExportCommand(dockerCli),
		newImportCommand(dockerCli),
		newRemoveCommand(dockerCli),
		newSetDockerEndpointCommand(dockerCli),
		newSetKubernetesEndpointCommand(dockerCli),
		newSetOptionsCommand(dockerCli),
	)
	return cmd
}
