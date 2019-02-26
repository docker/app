package commands

import (
	"github.com/docker/app/internal/packager"
	"github.com/docker/app/render"
	"github.com/docker/app/types"
	"github.com/docker/cli/cli"
	cliopts "github.com/docker/cli/opts"
	"github.com/spf13/cobra"
)

var (
	validateParametersFile []string
	validateEnv            []string
)

func validateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate [<app-name>] [-s key=value...] [-f parameters-file...]",
		Short: "Checks the rendered application is syntactically correct",
		Args:  cli.RequiresMaxArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := packager.Extract(firstOrEmpty(args),
				types.WithParametersFiles(validateParametersFile...),
			)
			if err != nil {
				return err
			}
			defer app.Cleanup()
			argParameters := cliopts.ConvertKVStringsToMap(validateEnv)
			_, err = render.Render(app, argParameters, nil)
			return err
		},
	}
	cmd.Flags().StringArrayVarP(&validateParametersFile, "parameters-files", "f", []string{}, "Override with parameters from files")
	cmd.Flags().StringArrayVarP(&validateEnv, "set", "s", []string{}, "Override parameters values")
	return cmd
}
