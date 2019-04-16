package commands

import (
	"github.com/docker/app/internal/packager"
	"github.com/docker/app/render"
	"github.com/docker/app/types"
	"github.com/docker/cli/cli"
	cliopts "github.com/docker/cli/opts"
	"github.com/spf13/cobra"
)

type validateOptions struct {
	parametersOptions
}

func validateCmd() *cobra.Command {
	var opts validateOptions
	cmd := &cobra.Command{
		Use:   "validate [<app-name>] [-s key=value...] [-f parameters-file...]",
		Short: "Checks the rendered application is syntactically correct",
		Args:  cli.RequiresMaxArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := packager.Extract(firstOrEmpty(args),
				types.WithParametersFiles(opts.parametersFiles...),
			)
			if err != nil {
				return err
			}
			defer app.Cleanup()
			argParameters := cliopts.ConvertKVStringsToMap(opts.overrides)
			_, err = render.Render(app, argParameters, nil)
			return err
		},
	}
	opts.parametersOptions.addFlags(cmd.Flags())
	return cmd
}
