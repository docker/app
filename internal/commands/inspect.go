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
		Use:   "inspect [<app-name>] [-s key=value...] [-f parameters-file...]",
		Short: "Shows metadata, parameters and a summary of the compose file for a given application",
		Args:  cli.RequiresMaxArgs(1),
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
	a, c, errBuf, err := prepareCustomAction(internal.ActionInspectName, dockerCli, appname, nil, opts.registryOptions, opts.pullOptions, opts.parametersOptions)
	if err != nil {
		return err
	}
	if err := a.Run(c, nil, nil); err != nil {
		return fmt.Errorf("inspect failed: %s", errBuf)
	}
	return nil
}
