package main

import (
	"github.com/docker/app/internal/inspect"
	"github.com/docker/app/internal/packager"
	"github.com/docker/app/types"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	cliopts "github.com/docker/cli/opts"
	"github.com/spf13/cobra"
)

var (
	inspectParametersFile []string
	inspectEnv            []string
)

func inspectCmd(dockerCli command.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "inspect [<app-name>] [-s key=value...] [-f parameters-file...]",
		Short: "Shows metadata, parameters and a summary of the compose file for a given application",
		Args:  cli.RequiresMaxArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := packager.Extract(firstOrEmpty(args),
				types.WithParametersFiles(inspectParametersFile...),
			)
			if err != nil {
				return err
			}
			defer app.Cleanup()
			argParameters := cliopts.ConvertKVStringsToMap(inspectEnv)
			return inspect.Inspect(dockerCli.Out(), app, argParameters, nil)
		},
	}
	cmd.Flags().StringArrayVarP(&inspectParametersFile, "parameters-files", "f", []string{}, "Override with parameters from files")
	cmd.Flags().StringArrayVarP(&inspectEnv, "set", "s", []string{}, "Override parameters values")
	return cmd
}
