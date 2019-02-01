package main

import (
	"github.com/deislabs/duffle/pkg/action"
	"github.com/deislabs/duffle/pkg/claim"
	"github.com/docker/app/types/parameters"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	cliopts "github.com/docker/cli/opts"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	inspectParametersFile []string
	inspectEnv            []string
)

// inspectCmd represents the inspect command
func inspectCmd(dockerCli command.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "inspect [<app-name>] [-s key=value...] [-f parameters-file...]",
		Short: "Shows metadata, parameters and a summary of the compose file for a given application",
		Args:  cli.RequiresMaxArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			muteDockerCli(dockerCli)
			appname := firstOrEmpty(args)
			bundle, err := resolveBundle(dockerCli, "", appname)
			if err != nil {
				return err
			}
			params, err := parameters.LoadFiles(inspectParametersFile)
			if err != nil {
				return err
			}
			overrides, err := parameters.FromFlatten(cliopts.ConvertKVStringsToMap(inspectEnv))
			if err != nil {
				return err
			}
			if params, err = parameters.Merge(params, overrides); err != nil {
				return err
			}
			c, err := claim.New("inspect")
			if err != nil {
				return err
			}
			driverImpl, err := prepareDriver(dockerCli)
			if err != nil {
				return err
			}
			c.Bundle = bundle
			c.Parameters = stringsKVToStringInterface(params.Flatten())

			a := &action.RunCustom{
				Action: "inspect",
				Driver: driverImpl,
			}
			err = a.Run(c, map[string]string{"docker.context": ""}, dockerCli.Out())
			return errors.Wrap(err, "Inspect failed")
		},
	}
	cmd.Flags().StringArrayVarP(&inspectParametersFile, "parameters-files", "f", []string{}, "Override with parameters from files")
	cmd.Flags().StringArrayVarP(&inspectEnv, "set", "s", []string{}, "Override parameters values")
	return cmd
}
