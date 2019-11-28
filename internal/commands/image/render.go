package image

import (
	"fmt"
	"io"
	"os"

	"github.com/deislabs/cnab-go/action"
	"github.com/deislabs/cnab-go/driver"
	"github.com/docker/app/internal"
	bdl "github.com/docker/app/internal/bundle"
	"github.com/docker/app/internal/cliopts"
	"github.com/docker/app/internal/cnab"
	"github.com/docker/app/internal/packager"
	appstore "github.com/docker/app/internal/store"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/config"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type renderOptions struct {
	cliopts.ParametersOptions
	formatDriver string
	renderOutput string
}

func renderCmd(dockerCli command.Cli, installerContext *cliopts.InstallerContextOptions) *cobra.Command {
	var opts renderOptions
	cmd := &cobra.Command{
		Use:     "render [OPTIONS] APP_IMAGE",
		Short:   "Render the Compose file for an App image",
		Example: `$ docker app render myrepo/myapp:1.0.0 --set key=value --parameters-file myparam.yml`,
		Args:    cli.ExactArgs(1),
		Hidden:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRender(dockerCli, args[0], opts, installerContext)
		},
	}
	opts.ParametersOptions.AddFlags(cmd.Flags())
	cmd.Flags().StringVarP(&opts.renderOutput, "output", "o", "-", "Output file")
	cmd.Flags().StringVar(&opts.formatDriver, "formatter", "yaml", "Configure the output format (yaml|json)")

	return cmd
}

func runRender(dockerCli command.Cli, appname string, opts renderOptions, installerContext *cliopts.InstallerContextOptions) error {
	var w io.Writer = os.Stdout
	if opts.renderOutput != "-" {
		f, err := os.Create(opts.renderOutput)
		if err != nil {
			return err
		}
		defer f.Close()
		w = f
	}

	s, err := appstore.NewApplicationStore(config.Dir())
	if err != nil {
		return err
	}
	imageStore, err := s.ImageStore()
	if err != nil {
		return err
	}
	img, ref, err := cnab.GetBundle(dockerCli, imageStore, appname)
	if err != nil {
		return errors.Wrapf(err, "could not render %q: no such App image", appname)
	}
	if err := packager.CheckAppVersion(dockerCli.Err(), img.Bundle); err != nil {
		return err
	}
	installation, err := appstore.NewInstallation("custom-action", ref.String(), img)
	if err != nil {
		return err
	}

	if err := bdl.MergeBundleParameters(installation,
		bdl.WithFileParameters(opts.ParametersFiles),
		bdl.WithCommandLineParameters(opts.Overrides),
	); err != nil {
		return err
	}

	defer muteDockerCli(dockerCli)()
	driverImpl, errBuf, err := cnab.SetupDriver(installation, dockerCli, installerContext, w)
	if err != nil {
		return err
	}
	action := &action.RunCustom{
		Action: internal.ActionRenderName,
		Driver: driverImpl,
	}
	installation.Parameters[internal.ParameterRenderFormatName] = opts.formatDriver

	cfgFunc := func(op *driver.Operation) error {
		op.Out = w
		return nil
	}

	if err := action.Run(&installation.Claim, nil, cfgFunc, cnab.WithRelocationMap(installation)); err != nil {
		return fmt.Errorf("render failed: %s\n%s", err, errBuf)
	}
	return nil
}
