package image

import (
	"fmt"
	"io/ioutil"

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
}

func firstOrEmpty(list []string) string {
	if len(list) != 0 {
		return list[0]
	}
	return ""
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
		Use:   "inspect [APP_NAME] [OPTIONS]",
		Short: "Shows metadata, parameters and a summary of the Compose file for a given application",
		Example: `$ docker app inspect my-installed-app
$docker app inspect my-app:1.0.0`,
		Args: cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInspect(dockerCli, firstOrEmpty(args), opts)
		},
	}
	cmd.Flags().BoolVar(&opts.pretty, "pretty", false, "Pretty print the output")

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

	driverImpl, errBuf := cnab.PrepareDriver(dockerCli, cnab.BindMount{}, nil)
	a := &action.RunCustom{
		Action: internal.ActionInspectName,
		Driver: driverImpl,
	}

	format := "json"
	if opts.pretty {
		format = "pretty"
	}

	installation.SetParameter(internal.ParameterInspectFormatName, format)

	if err := a.Run(&installation.Claim, nil, nil); err != nil {
		return fmt.Errorf("inspect failed: %s\n%s", err, errBuf)
	}
	return nil
}
