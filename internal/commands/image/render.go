package image

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/deislabs/cnab-go/driver"

	"github.com/deislabs/cnab-go/action"
	"github.com/docker/app/internal"
	bdl "github.com/docker/app/internal/bundle"
	"github.com/docker/app/internal/cliopts"
	"github.com/docker/app/internal/cnab"
	appstore "github.com/docker/app/internal/store"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/config"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type renderOptions struct {
	cliopts.ParametersOptions
	cliopts.InstallerContextOptions
	formatDriver string
	renderOutput string
}

func renderCmd(dockerCli command.Cli) *cobra.Command {
	var opts renderOptions
	cmd := &cobra.Command{
		Use:     "render [OPTIONS] APP_IMAGE",
		Short:   "Render the Compose file for an App image",
		Example: `$ docker app render myrepo/myapp:1.0.0 --set key=value --parameters-file myparam.yml`,
		Args:    cli.ExactArgs(1),
		Hidden:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRender(dockerCli, args[0], opts)
		},
	}
	opts.ParametersOptions.AddFlags(cmd.Flags())
	opts.InstallerContextOptions.AddFlags(cmd.Flags())
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

	cfgFunc := func(op *driver.Operation) error {
		op.Out = w
		return nil
	}

	action, installation, errBuf, err := prepareCustomAction(internal.ActionRenderName, dockerCli, appname, w, opts)
	if err != nil {
		return err
	}
	installation.Parameters[internal.ParameterRenderFormatName] = opts.formatDriver

	if err := action.Run(&installation.Claim, nil, cfgFunc); err != nil {
		return fmt.Errorf("render failed: %s\n%s", err, errBuf)
	}
	return nil
}

func prepareCustomAction(actionName string, dockerCli command.Cli, appname string, stdout io.Writer, opts renderOptions) (*action.RunCustom, *appstore.Installation, *bytes.Buffer, error) {
	s, err := appstore.NewApplicationStore(config.Dir())
	if err != nil {
		return nil, nil, nil, err
	}
	bundleStore, err := s.BundleStore()
	if err != nil {
		return nil, nil, nil, err
	}
	bundle, ref, err := cnab.GetBundle(dockerCli, bundleStore, appname)
	if err != nil {
		return nil, nil, nil, errors.Wrapf(err, "could not render %q: no such App image", appname)
	}
	installation, err := appstore.NewInstallation("custom-action", ref.String())
	if err != nil {
		return nil, nil, nil, err
	}
	installation.Bundle = bundle.Bundle

	if err := bdl.MergeBundleParameters(installation,
		bdl.WithFileParameters(opts.ParametersFiles),
		bdl.WithCommandLineParameters(opts.Overrides),
	); err != nil {
		return nil, nil, nil, err
	}

	driverImpl, errBuf, err := cnab.SetupDriver(installation, dockerCli, opts.InstallerContextOptions, stdout)
	if err != nil {
		return nil, nil, nil, err
	}
	a := &action.RunCustom{
		Action: actionName,
		Driver: driverImpl,
	}
	return a, installation, errBuf, nil
}
