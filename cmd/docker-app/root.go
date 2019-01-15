package main

import (
	"fmt"

	"github.com/docker/app/internal"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	cliconfig "github.com/docker/cli/cli/config"
	"github.com/docker/cli/cli/debug"
	cliflags "github.com/docker/cli/cli/flags"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// rootCmd represents the base command when called without any subcommands
// FIXME(vdemeester) use command.Cli interface
func newRootCmd(dockerCli *command.DockerCli) *cobra.Command {
	opts := cliflags.NewClientOptions()
	var flags *pflag.FlagSet

	cmd := &cobra.Command{
		Use:          "docker-app",
		Short:        "Docker Application Packages",
		Long:         `Build and deploy Docker Application Packages.`,
		SilenceUsage: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			opts.Common.SetDefaultOptions(flags)
			dockerPreRun(opts)
			return dockerCli.Initialize(opts)
		},
		Version: fmt.Sprintf("%s, build %s", internal.Version, internal.GitCommit),
	}
	cli.SetupRootCommand(cmd)
	flags = cmd.Flags()
	flags.BoolP("version", "v", false, "Print version information")
	opts.Common.InstallFlags(flags)
	cmd.SetVersionTemplate("docker-app version {{.Version}}\n")
	addCommands(cmd, dockerCli)
	return cmd
}

// addCommands adds all the commands from cli/command to the root command
func addCommands(cmd *cobra.Command, dockerCli command.Cli) {
	cmd.AddCommand(
		deployCmd(dockerCli),
		initCmd(),
		inspectCmd(dockerCli),
		mergeCmd(dockerCli),
		pushCmd(),
		renderCmd(dockerCli),
		splitCmd(),
		validateCmd(),
		versionCmd(dockerCli),
		completionCmd(dockerCli, cmd),
		bundleCmd(dockerCli),
	)
	if internal.Experimental == "on" {
		cmd.AddCommand(
			imageAddCmd(),
			imageLoadCmd(),
			packCmd(dockerCli),
			pullCmd(),
			unpackCmd(),
		)
	}
}

func firstOrEmpty(list []string) string {
	if len(list) != 0 {
		return list[0]
	}
	return ""
}

func dockerPreRun(opts *cliflags.ClientOptions) {
	cliflags.SetLogLevel(opts.Common.LogLevel)

	if opts.ConfigDir != "" {
		cliconfig.SetDir(opts.ConfigDir)
	}

	if opts.Common.Debug {
		debug.Enable()
	}
}
