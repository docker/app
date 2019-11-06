package cliopts

import (
	"fmt"
	"os"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/config"
	"github.com/docker/cli/cli/flags"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
)

type InstallerContextOptions struct {
	installerContext string
}

func (o *InstallerContextOptions) AddFlags(flags *pflag.FlagSet) {
	defaultContext, ok := os.LookupEnv("DOCKER_INSTALLER_CONTEXT")
	if !ok {
		defaultContext = "default"
	}
	flags.StringVar(&o.installerContext, "installer-context", defaultContext, "Context on which the installer image is ran")
}

func (o *InstallerContextOptions) SetInstallerContext(dockerCli command.Cli) (command.Cli, error) {
	if o.installerContext != dockerCli.CurrentContext() {
		if _, err := dockerCli.ContextStore().GetMetadata(o.installerContext); err != nil {
			return nil, errors.Wrapf(err, "Unknown docker context %s", o.installerContext)
		}
		fmt.Fprintf(dockerCli.Out(), "Using context %q to run installer image", o.installerContext)
		cli, err := command.NewDockerCli()
		if err != nil {
			return nil, err
		}
		opts := flags.ClientOptions{
			Common: &flags.CommonOptions{
				Context:  o.installerContext,
				LogLevel: logrus.GetLevel().String(),
			},
			ConfigDir: config.Dir(),
		}
		if err = cli.Apply(
			command.WithInputStream(dockerCli.In()),
			command.WithOutputStream(dockerCli.Out()),
			command.WithErrorStream(dockerCli.Err())); err != nil {
			return nil, err
		}
		if err = cli.Initialize(&opts); err != nil {
			return nil, err
		}
		return cli, nil
	}
	return dockerCli, nil
}
