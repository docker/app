package commands

import (
	"fmt"
	"os"

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
		Use:   "validate [APP_NAME] [--set KEY=VALUE ...] [--parameters-file PARAMETERS_FILE]",
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
			if err != nil {
				return err
			}
			fmt.Fprintf(os.Stdout, "Validated %q\n", app.Path)
			return nil
		},
	}
	opts.parametersOptions.addFlags(cmd.Flags())
	return cmd
}
