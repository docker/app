package commands

import (
	"io/ioutil"

	"github.com/docker/cli/cli/command"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// NewRootCmd returns the base root command.
func NewRootCmd(use string, dockerCli command.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Short: "Docker Application Packages",
		Long:  `Build and deploy Docker Application Packages.`,
		Use:   use,
	}
	addCommands(cmd, dockerCli)
	return cmd
}

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

func muteDockerCli(dockerCli command.Cli) func() {
	stdout := dockerCli.Out()
	stderr := dockerCli.Err()
	dockerCli.Apply(command.WithCombinedStreams(ioutil.Discard))
	return func() {
		dockerCli.Apply(command.WithOutputStream(stdout), command.WithErrorStream(stderr))
	}
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
	flags.StringSliceVar(&o.insecureRegistries, "insecure-registries", nil, "Use HTTP instead of HTTPS when pulling from/pushing to those registries")
}

type pullOptions struct {
	pull bool
}

func (o *pullOptions) addFlags(flags *pflag.FlagSet) {
	flags.BoolVar(&o.pull, "pull", false, "Pull the bundle")
}
