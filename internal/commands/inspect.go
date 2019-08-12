package commands

import (
	"fmt"

	"github.com/docker/app/internal"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/spf13/cobra"
)

type inspectOptions struct {
	parametersOptions
	registryOptions
	pullOptions
}

func inspectCmd(dockerCli command.Cli) *cobra.Command {
	var opts inspectOptions
	cmd := &cobra.Command{
		Use:     "inspect [APP_NAME] [OPTIONS]",
		Short:   "Shows metadata, parameters and a summary of the Compose file for a given application",
		Example: `$ docker app inspect myapp.dockerapp`,
		Args:    cli.RequiresMaxArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInspect(dockerCli, firstOrEmpty(args), opts)
		},
	}
	opts.parametersOptions.addFlags(cmd.Flags())
	opts.registryOptions.addFlags(cmd.Flags())
	opts.pullOptions.addFlags(cmd.Flags())
	return cmd
}

func runInspect(dockerCli command.Cli, appname string, opts inspectOptions) error {
	defer muteDockerCli(dockerCli)()
	action, installation, errBuf, err := prepareCustomAction(internal.ActionInspectName, dockerCli, appname, nil, opts.registryOptions, opts.pullOptions, opts.parametersOptions)
	if err != nil {
		return err
	}
	if err := action.Run(&installation.Claim, nil, nil); err != nil {
		return fmt.Errorf("inspect failed: %s\n%s", err, errBuf)
	}
	return nil
}
