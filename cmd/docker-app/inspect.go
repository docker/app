package main

import (
	"github.com/deislabs/duffle/pkg/action"
	"github.com/deislabs/duffle/pkg/claim"
	"github.com/docker/app/internal"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func inspectCmd(dockerCli command.Cli) *cobra.Command {
	var opts parametersOptions
	cmd := &cobra.Command{
		Use:   "inspect [<app-name>] [-s key=value...] [-f parameters-file...]",
		Short: "Shows metadata, parameters and a summary of the compose file for a given application",
		Args:  cli.RequiresMaxArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			defer muteDockerCli(dockerCli)()
			appname := firstOrEmpty(args)

			c, err := claim.New("inspect")
			if err != nil {
				return err
			}
			driverImpl, err := prepareDriver(dockerCli)
			if err != nil {
				return err
			}
			bundle, err := resolveBundle(dockerCli, appname)
			if err != nil {
				return err
			}
			c.Bundle = bundle

			parameters, err := mergeBundleParameters(c.Bundle,
				withFileParameters(opts.parametersFiles),
				withCommandLineParameters(opts.overrides),
			)
			if err != nil {
				return err
			}
			c.Parameters = parameters

			a := &action.RunCustom{
				Action: internal.Namespace + "inspect",
				Driver: driverImpl,
			}
			err = a.Run(c, map[string]string{"docker.context": ""}, dockerCli.Out())
			return errors.Wrap(err, "Inspect failed")
		},
	}
	opts.addFlags(cmd.Flags())
	return cmd
}
