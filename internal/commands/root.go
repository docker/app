package commands

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/docker/app/internal"
	"github.com/docker/app/internal/commands/build"
	"github.com/docker/app/internal/commands/image"
	"github.com/docker/app/internal/store"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/config"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	showVersion bool
)

// NewRootCmd returns the base root command.
func NewRootCmd(use string, dockerCli command.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Short:       "Docker Application",
		Long:        `A tool to build and manage Docker Applications.`,
		Use:         use,
		Annotations: map[string]string{"experimentalCLI": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if showVersion {
				fmt.Fprintln(os.Stdout, internal.FullVersion()) //nolint:errcheck
				return nil
			}

			if len(args) != 0 {
				return fmt.Errorf("%q is not a docker app command\nSee 'docker app --help'", args[0])
			}
			cmd.HelpFunc()(cmd, args)
			return nil
		},
	}
	addCommands(cmd, dockerCli)

	cmd.Flags().BoolVar(&showVersion, "version", false, "Print version information")
	return cmd
}

func addCommands(cmd *cobra.Command, dockerCli command.Cli) {
	cmd.AddCommand(
		runCmd(dockerCli),
		updateCmd(dockerCli),
		removeCmd(dockerCli),
		listCmd(dockerCli),
		initCmd(dockerCli),
		inspectCmd(dockerCli),
		renderCmd(dockerCli),
		validateCmd(),
		pushCmd(dockerCli),
		pullCmd(dockerCli),
		image.Cmd(dockerCli),
		build.Cmd(dockerCli),
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
	dockerCli.Apply(command.WithCombinedStreams(ioutil.Discard)) //nolint:errcheck // WithCombinedStreams cannot error
	return func() {
		dockerCli.Apply(command.WithOutputStream(stdout), command.WithErrorStream(stderr)) //nolint:errcheck // as above
	}
}

func prepareStores(targetContext string) (store.BundleStore, store.InstallationStore, store.CredentialStore, error) {
	appstore, err := store.NewApplicationStore(config.Dir())
	if err != nil {
		return nil, nil, nil, err
	}
	installationStore, err := appstore.InstallationStore(targetContext)
	if err != nil {
		return nil, nil, nil, err
	}
	bundleStore, err := appstore.BundleStore()
	if err != nil {
		return nil, nil, nil, err
	}
	credentialStore, err := appstore.CredentialStore(targetContext)
	if err != nil {
		return nil, nil, nil, err
	}
	return bundleStore, installationStore, credentialStore, nil
}

func prepareBundleStore() (store.BundleStore, error) {
	appstore, err := store.NewApplicationStore(config.Dir())
	if err != nil {
		return nil, err
	}
	bundleStore, err := appstore.BundleStore()
	if err != nil {
		return nil, err
	}
	return bundleStore, nil
}

type parametersOptions struct {
	parametersFiles []string
	overrides       []string
}

func (o *parametersOptions) addFlags(flags *pflag.FlagSet) {
	flags.StringArrayVar(&o.parametersFiles, "parameters-file", []string{}, "Override parameters file")
	flags.StringArrayVarP(&o.overrides, "set", "s", []string{}, "Override parameter value")
}

type credentialOptions struct {
	targetContext    string
	credentialsets   []string
	credentials      []string
	sendRegistryAuth bool
}

func (o *credentialOptions) addFlags(flags *pflag.FlagSet) {
	flags.StringVar(&o.targetContext, "target-context", "", "Context on which the application is installed (default: <current-context>)")
	flags.StringArrayVar(&o.credentialsets, "credential-set", []string{}, "Use a YAML file containing a credential set or a credential set present in the credential store")
	flags.StringArrayVar(&o.credentials, "credential", nil, "Add a single credential, additive ontop of any --credential-set used")
	flags.BoolVar(&o.sendRegistryAuth, "with-registry-auth", false, "Sends registry auth")
}

func (o *credentialOptions) SetDefaultTargetContext(dockerCli command.Cli) {
	o.targetContext = getTargetContext(o.targetContext, dockerCli.CurrentContext())
}

func (o *credentialOptions) CredentialSetOpts(dockerCli command.Cli, credentialStore store.CredentialStore) []credentialSetOpt {
	return []credentialSetOpt{
		addNamedCredentialSets(credentialStore, o.credentialsets),
		addCredentials(o.credentials),
		addDockerCredentials(o.targetContext, dockerCli.ContextStore()),
		addRegistryCredentials(o.sendRegistryAuth, dockerCli),
	}
}

// insecureRegistriesFromEngine reads the registry configuration from the daemon and returns
// a list of all insecure ones.
func insecureRegistriesFromEngine(dockerCli command.Cli) ([]string, error) {
	registries := []string{}

	info, err := dockerCli.Client().Info(context.Background())
	if err != nil {
		return nil, fmt.Errorf("could not get docker info: %v", err)
	}

	for _, reg := range info.RegistryConfig.IndexConfigs {
		if !reg.Secure {
			registries = append(registries, reg.Name)
		}
	}

	logrus.Debugf("insecure registries: %v", registries)

	return registries, nil
}
