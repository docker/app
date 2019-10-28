package image

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/docker/app/internal/cliopts"

	"github.com/deislabs/cnab-go/action"
	"github.com/docker/app/internal"
	"github.com/docker/app/internal/cnab"
	appstore "github.com/docker/app/internal/store"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/config"
	"github.com/spf13/cobra"
)

type inspectOptions struct {
	pretty bool
	cliopts.InstallerContextOptions
}

func muteDockerCli(dockerCli command.Cli) func() {
	stdout := dockerCli.Out()
	stderr := dockerCli.Err()
	dockerCli.Apply(command.WithCombinedStreams(ioutil.Discard)) //nolint:errcheck // WithCombinedStreams cannot error
	return func() {
		dockerCli.Apply(command.WithOutputStream(stdout), command.WithErrorStream(stderr)) //nolint:errcheck // as above
	}
}

func inspectCmd(dockerCli command.Cli) *cobra.Command {
	var opts inspectOptions
	cmd := &cobra.Command{
		Use:   "inspect [OPTIONS] APP_IMAGE",
		Short: "Display detailed information about an App image",
		Example: `$ docker app image inspect myapp
$ docker app image inspect myapp:1.0.0
$ docker app image inspect myrepo/myapp:1.0.0
$ docker app image inspect 34be4a0c5f50`,
		Args: cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInspect(dockerCli, args[0], opts)
		},
	}
	opts.InstallerContextOptions.AddFlags(cmd.Flags())
	cmd.Flags().BoolVar(&opts.pretty, "pretty", false, "Print the information in a human friendly format")

	return cmd
}

func runInspect(dockerCli command.Cli, appname string, opts inspectOptions) error {
	defer muteDockerCli(dockerCli)()
	s, err := appstore.NewApplicationStore(config.Dir())
	if err != nil {
		return err
	}
	bundleStore, err := s.BundleStore()
	if err != nil {
		return err
	}
	bndl, ref, err := cnab.GetBundle(dockerCli, bundleStore, appname)

	if err != nil {
		return err
	}
	installation, err := appstore.NewInstallation("custom-action", ref.String())
	if err != nil {
		return err
	}
	installation.Bundle = bndl
	driverImpl, errBuf, err := cnab.SetupDriver(installation, dockerCli, opts.InstallerContextOptions, os.Stdout)
	if err != nil {
		return err
	}
	a := &action.RunCustom{
		Action: internal.ActionInspectName,
		Driver: driverImpl,
	}

	format := "json"
	if opts.pretty {
		format = "pretty"
	}

	installation.SetParameter(internal.ParameterInspectFormatName, format)

	if err := a.Run(&installation.Claim, nil); err != nil {
		return fmt.Errorf("inspect failed: %s\n%s", err, errBuf)
	}
	return nil
}
