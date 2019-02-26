package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/docker/app/internal"
	app "github.com/docker/app/internal/commands"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	cliflags "github.com/docker/cli/cli/flags"
	"github.com/sirupsen/logrus"
)

func main() {
	dockerCli, err := command.NewDockerCli()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	logrus.SetOutput(dockerCli.Err())

	cmd := app.NewRootCmd("docker-app", dockerCli)
	configureRootCmd(cmd, dockerCli)

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func configureRootCmd(cmd *cobra.Command, dockerCli *command.DockerCli) {
	var (
		opts  *cliflags.ClientOptions
		flags *pflag.FlagSet
	)

	cmd.SilenceUsage = true
	cmd.TraverseChildren = true
	cmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		opts.Common.SetDefaultOptions(flags)
		return dockerCli.Initialize(opts)
	}
	cmd.Version = fmt.Sprintf("%s, build %s", internal.Version, internal.GitCommit)

	opts, flags, _ = cli.SetupRootCommand(cmd)
	flags.BoolP("version", "v", false, "Print version information")
	cmd.SetVersionTemplate("docker-app version {{.Version}}\n")
}
