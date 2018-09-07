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
	inspectSettingsFile []string
	inspectEnv          []string
)

// inspectCmd represents the inspect command
func inspectCmd(dockerCli command.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "inspect [<app-name>] [-s key=value...] [-f settings-file...]",
		Short: "Shows metadata, settings and a summary of the compose file for a given application",
		Args:  cli.RequiresMaxArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := packager.Extract(firstOrEmpty(args),
				types.WithSettingsFiles(inspectSettingsFile...),
			)
			if err != nil {
				return err
			}
			defer app.Cleanup()
			argSettings := cliopts.ConvertKVStringsToMap(inspectEnv)
			return inspect.Inspect(dockerCli.Out(), app, argSettings)
		},
	}
	cmd.Flags().StringArrayVarP(&inspectSettingsFile, "settings-files", "f", []string{}, "Override settings files")
	cmd.Flags().StringArrayVarP(&inspectEnv, "set", "s", []string{}, "Override settings values")
	return cmd
}
