package commands

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/deislabs/cnab-go/action"
	"github.com/docker/app/internal"
	"github.com/docker/app/internal/cnab"
	appstore "github.com/docker/app/internal/store"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/config"
	"github.com/spf13/cobra"
)

type renderOptions struct {
	parametersOptions
	formatDriver string
	renderOutput string
}

func renderCmd(dockerCli command.Cli) *cobra.Command {
	var opts renderOptions
	cmd := &cobra.Command{
		Use:     "render [OPTIONS] APP_IMAGE",
		Short:   "Render the Compose file for an App image",
		Example: `$ docker app render myrepo/myapp:1.0.0 --set key=value --parameters-file myparam.yml`,
		Args:    cli.RequiresMaxArgs(1),
		Hidden:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRender(dockerCli, firstOrEmpty(args), opts)
		},
	}
	opts.parametersOptions.addFlags(cmd.Flags())
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

	action, installation, errBuf, err := prepareCustomAction(internal.ActionRenderName, dockerCli, appname, w, opts.parametersOptions)
	if err != nil {
		return err
	}
	installation.Parameters[internal.ParameterRenderFormatName] = opts.formatDriver

	if err := action.Run(&installation.Claim, nil, nil); err != nil {
		return fmt.Errorf("render failed: %s\n%s", err, errBuf)
	}
	return nil
}

func prepareCustomAction(actionName string, dockerCli command.Cli, appname string, stdout io.Writer, paramsOpts parametersOptions) (*action.RunCustom, *appstore.Installation, *bytes.Buffer, error) {
	s, err := appstore.NewApplicationStore(config.Dir())
	if err != nil {
		return nil, nil, nil, err
	}
	bundleStore, err := s.BundleStore()
	if err != nil {
		return nil, nil, nil, err
	}
	bundle, ref, err := cnab.ResolveBundle(dockerCli, bundleStore, appname)
	if err != nil {
		return nil, nil, nil, err
	}
	installation, err := appstore.NewInstallation("custom-action", ref)
	if err != nil {
		return nil, nil, nil, err
	}
	installation.Bundle = bundle

	if err := mergeBundleParameters(installation,
		withFileParameters(paramsOpts.parametersFiles),
		withCommandLineParameters(paramsOpts.overrides),
	); err != nil {
		return nil, nil, nil, err
	}

	driverImpl, errBuf := cnab.PrepareDriver(dockerCli, cnab.BindMount{}, stdout)
	a := &action.RunCustom{
		Action: actionName,
		Driver: driverImpl,
	}
	return a, installation, errBuf, nil
}
