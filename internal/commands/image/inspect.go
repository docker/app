package image

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/docker/app/internal/packager"

	"github.com/deislabs/cnab-go/action"
	"github.com/docker/app/internal"
	"github.com/docker/app/internal/cliopts"
	"github.com/docker/app/internal/cnab"
	"github.com/docker/app/internal/inspect"
	appstore "github.com/docker/app/internal/store"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/config"
	"github.com/spf13/cobra"
)

const inspectExample = `- $ docker app image inspect myapp
- $ docker app image inspect myapp:1.0.0
- $ docker app image inspect myrepo/myapp:1.0.0
- $ docker app image inspect 34be4a0c5f50`

type inspectOptions struct {
	pretty bool
}

func muteDockerCli(dockerCli command.Cli) func() {
	stdout := dockerCli.Out()
	stderr := dockerCli.Err()
	dockerCli.Apply(command.WithCombinedStreams(ioutil.Discard)) //nolint:errcheck // WithCombinedStreams cannot error
	return func() {
		dockerCli.Apply(command.WithOutputStream(stdout), command.WithErrorStream(stderr)) //nolint:errcheck // as above
	}
}

func inspectCmd(dockerCli command.Cli, installerContext *cliopts.InstallerContextOptions) *cobra.Command {
	var opts inspectOptions
	cmd := &cobra.Command{
		Use:     "inspect [OPTIONS] APP_IMAGE",
		Short:   "Display detailed information about an App image",
		Example: inspectExample,
		Args:    cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInspect(dockerCli, args[0], opts, installerContext)
		},
	}
	cmd.Flags().BoolVar(&opts.pretty, "pretty", false, "Print the information in a human friendly format")

	return cmd
}

func runInspect(dockerCli command.Cli, appname string, opts inspectOptions, installerContext *cliopts.InstallerContextOptions) error {
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
	if err := packager.CheckAppVersion(dockerCli.Err(), bndl.Bundle); err != nil {
		return err
	}

	format := "json"
	if opts.pretty {
		format = "pretty"
	}

	installation, err := appstore.NewInstallation("custom-action", ref.String(), bndl)
	if err != nil {
		return err
	}

	if _, hasAction := installation.Bundle.Actions[internal.ActionInspectName]; hasAction {
		driverImpl, errBuf, err := cnab.SetupDriver(installation, dockerCli, installerContext, os.Stdout)
		if err != nil {
			return err
		}
		a := &action.RunCustom{
			Action: internal.ActionInspectName,
			Driver: driverImpl,
		}

		installation.SetParameter(internal.ParameterInspectFormatName, format)
		if err = a.Run(&installation.Claim, nil, cnab.WithRelocationMap(installation)); err != nil {
			return fmt.Errorf("inspect failed: %s\n%s", err, errBuf)
		}
	} else {
		if err = inspect.ImageInspectCNAB(os.Stdout, bndl.Bundle, format); err != nil {
			return fmt.Errorf("inspect failed: %s", err)
		}
	}
	return nil
}
