package commands

import (
	"io"
	"os"

	"github.com/deislabs/duffle/pkg/action"
	"github.com/deislabs/duffle/pkg/claim"
	"github.com/docker/app/internal"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type renderOptions struct {
	parametersOptions
	registryOptions
	pullOptions

	formatDriver string
	renderOutput string
}

func renderCmd(dockerCli command.Cli) *cobra.Command {
	var opts renderOptions
	cmd := &cobra.Command{
		Use:   "render <app-name> [-s key=value...] [-f parameters-file...]",
		Short: "Render the Compose file for the application",
		Long:  `Render the Compose file for the application.`,
		Args:  cli.RequiresMaxArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRender(dockerCli, firstOrEmpty(args), opts)
		},
	}
	opts.parametersOptions.addFlags(cmd.Flags())
	opts.registryOptions.addFlags(cmd.Flags())
	opts.pullOptions.addFlags(cmd.Flags())
	cmd.Flags().StringVarP(&opts.renderOutput, "output", "o", "-", "Output file")
	cmd.Flags().StringVar(&opts.formatDriver, "formatter", "yaml", "Configure the output format (yaml|json)")

	return cmd
}

func runRender(dockerCli command.Cli, appname string, opts renderOptions) error {
	defer muteDockerCli(dockerCli)()

	c, err := claim.New("render")
	if err != nil {
		return err
	}
	driverImpl, err := prepareDriver(dockerCli, bindMount{})
	if err != nil {
		return err
	}
	bundle, err := resolveBundle(dockerCli, appname, opts.pull, opts.insecureRegistries)
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
	c.Parameters[internal.Namespace+"render-format"] = opts.formatDriver

	a := &action.RunCustom{
		Action: internal.Namespace + "render",
		Driver: driverImpl,
	}

	var writer io.Writer = dockerCli.Out()
	if opts.renderOutput != "-" {
		f, err := os.Create(opts.renderOutput)
		if err != nil {
			return err
		}
		defer f.Close()
		writer = f
	}
	err = a.Run(c, nil, writer)
	return errors.Wrap(err, "Render failed")
}
