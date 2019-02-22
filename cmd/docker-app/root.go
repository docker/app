package main

import (
	"fmt"
	"io/ioutil"

	"github.com/docker/app/internal"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	cliflags "github.com/docker/cli/cli/flags"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// rootCmd represents the base command when called without any subcommands
func newRootCmd(dockerCli *command.DockerCli) *cobra.Command {
	var (
		opts  *cliflags.ClientOptions
		flags *pflag.FlagSet
	)

	cmd := &cobra.Command{
		Use:              "docker-app",
		Short:            "Docker Application Packages",
		Long:             `Build and deploy Docker Application Packages.`,
		SilenceUsage:     true,
		TraverseChildren: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			opts.Common.SetDefaultOptions(flags)
			return dockerCli.Initialize(opts)
		},
		Version: fmt.Sprintf("%s, build %s", internal.Version, internal.GitCommit),
	}
	opts, flags, _ = cli.SetupRootCommand(cmd)
	flags.BoolP("version", "v", false, "Print version information")
	cmd.SetVersionTemplate("docker-app version {{.Version}}\n")
	addCommands(cmd, dockerCli)
	return cmd
}

// addCommands adds all the commands from cli/command to the root command
func addCommands(cmd *cobra.Command, dockerCli command.Cli) {
	cmd.AddCommand(
		installCmd(dockerCli),
		upgradeCmd(dockerCli),
		uninstallCmd(dockerCli),
		statusCmd(dockerCli),
		initCmd(),
		inspectCmd(dockerCli),
		mergeCmd(dockerCli),
		renderCmd(dockerCli),
		splitCmd(),
		validateCmd(),
		versionCmd(dockerCli),
		completionCmd(dockerCli, cmd),
		bundleCmd(dockerCli),
		pushCmd(dockerCli),
		pullCmd(dockerCli),
	)
}

func firstOrEmpty(list []string) string {
	if len(list) != 0 {
		return list[0]
	}
	return ""
}

func muteDockerCli(dockerCli command.Cli) {
	dockerCli.Apply(command.WithCombinedStreams(ioutil.Discard))
}

type parametersOptions struct {
	parametersFiles []string
	overrides       []string
}

func (o *parametersOptions) addFlags(flags *pflag.FlagSet) {
	flags.StringArrayVarP(&o.parametersFiles, "parameters-files", "f", []string{}, "Override parameters files")
	flags.StringArrayVarP(&o.overrides, "set", "s", []string{}, "Override parameters values")
}

type credentialOptions struct {
	targetContext  string
	credentialsets []string
}

func (o *credentialOptions) addFlags(flags *pflag.FlagSet) {
	flags.StringVar(&o.targetContext, "target-context", "", "Context on which the application is executed")
	flags.StringArrayVarP(&o.credentialsets, "credential-set", "c", []string{}, "Use a duffle credentialset (either a YAML file, or a credential set present in the duffle credential store)")
}

type registryOptions struct {
	insecureRegistries []string
}

func (o *registryOptions) addFlags(flags *pflag.FlagSet) {
	flags.StringSliceVar(&o.insecureRegistries, "insecure-registries", nil, "Use HTTP instead of HTTPS when pulling or pushing from/to those registries")
}
