package main

import (
	"github.com/docker/app/internal/packager"
	"github.com/docker/app/internal/validator"
	"github.com/docker/app/types"
	"github.com/docker/cli/cli"
	cliopts "github.com/docker/cli/opts"
	"github.com/spf13/cobra"
)

var (
	validateSettingsFile []string
	validateEnv          []string
)

func validateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate [<app-name>] [-s key=value...] [-f settings-file...]",
		Short: "Checks the rendered application is syntactically correct",
		Args:  cli.RequiresMaxArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := packager.Extract(firstOrEmpty(args),
				types.WithSettingsFiles(validateSettingsFile...),
			)
			if err != nil {
				return err
			}
			defer app.Cleanup()
			argSettings := cliopts.ConvertKVStringsToMap(validateEnv)
			return validator.Validate(app, argSettings)
		},
	}
	cmd.Flags().StringArrayVarP(&validateSettingsFile, "settings-files", "f", []string{}, "Override settings files")
	cmd.Flags().StringArrayVarP(&validateEnv, "set", "s", []string{}, "Override settings values")
	return cmd
}
