package commands

import (
	"fmt"
	"os"

	"github.com/docker/app/internal/cliopts"
	"github.com/docker/app/internal/compose"
	"github.com/docker/app/internal/packager"
	"github.com/docker/app/render"
	"github.com/docker/app/types"
	"github.com/docker/cli/cli"
	dockercliopts "github.com/docker/cli/opts"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type validateOptions struct {
	cliopts.ParametersOptions
}

func validateCmd() *cobra.Command {
	var opts validateOptions
	cmd := &cobra.Command{
		Use:         "validate [OPTIONS] APP_DEFINITION",
		Short:       "Check that an App definition (.dockerapp) is syntactically correct",
		Example:     `$ docker app validate myapp.dockerapp --set key=value --parameters-file myparam.yml`,
		Args:        cli.RequiresMaxArgs(1),
		Annotations: map[string]string{"experimentalCLI": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runValidate(args, opts)
		},
	}
	opts.ParametersOptions.AddFlags(cmd.Flags())
	return cmd
}

func runValidate(args []string, opts validateOptions) error {
	app, err := packager.Extract(firstOrEmpty(args),
		types.WithParametersFiles(opts.ParametersFiles...),
	)
	if err != nil {
		return err
	}
	defer app.Cleanup()
	argParameters := dockercliopts.ConvertKVStringsToMap(opts.Overrides)
	_, err = render.Render(app, argParameters, nil)
	if err != nil {
		return err
	}

	vars, err := compose.ExtractVariables(app.Composes()[0], compose.ExtrapolationPattern)
	if err != nil {
		return errors.Wrap(err, "failed to parse compose file")
	}
	for k := range app.Parameters().Flatten() {
		if _, ok := vars[k]; !ok {
			return fmt.Errorf("%s is declared as parameter but not used by the compose file", k)
		}
	}

	fmt.Fprintf(os.Stdout, "Validated %q\n", app.Path)
	return nil
}
