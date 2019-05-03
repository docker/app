package commands

import (
	"fmt"
	"io"
	"os"

	"github.com/docker/app/internal"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
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
		Use:     "render [APP_NAME] [--set KEY=VALUE ...] [--parameters-file PARAMETERS-FILE ...] [OPTIONS]",
		Short:   "Render the Compose file for an Application Package",
		Example: `$ docker app render myapp.dockerapp --set key=value`,
		Args:    cli.RequiresMaxArgs(1),
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

	var w io.Writer = os.Stdout
	if opts.renderOutput != "-" {
		f, err := os.Create(opts.renderOutput)
		if err != nil {
			return err
		}
		defer f.Close()
		w = f
	}

	action, installation, errBuf, err := prepareCustomAction(internal.ActionRenderName, dockerCli, appname, w, opts.registryOptions, opts.pullOptions, opts.parametersOptions)
	if err != nil {
		return err
	}
	installation.Parameters[internal.ParameterRenderFormatName] = opts.formatDriver

	fmt.Fprintf(os.Stderr, "Rendering %q using format %q\n", appname, opts.formatDriver)
	fmt.Fprintf(os.Stderr, "Action: %+v\n", action)
	fmt.Fprintf(os.Stderr, "Installation: %+v\n", installation)
	if err := action.Run(&installation.Claim, nil, nil); err != nil {
		return fmt.Errorf("render failed: %s", errBuf)
	}
	return nil
}
