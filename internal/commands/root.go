package commands

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/deislabs/cnab-go/claim"
	"github.com/deislabs/cnab-go/driver"
	"github.com/docker/app/internal"
	"github.com/docker/app/internal/cnab"
	"github.com/docker/app/internal/commands/build"
	"github.com/docker/app/internal/commands/image"
	"github.com/docker/app/internal/store"
	appstore "github.com/docker/app/internal/store"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/config"
	"github.com/docker/cli/cli/flags"
	"github.com/pkg/errors"
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
		Short:       "Docker App",
		Long:        `A tool to build, share and run a Docker App`,
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

func setupDriver(installation *store.Installation, dockerCli command.Cli, opts installerContextOptions) (driver.Driver, *bytes.Buffer, error) {
	dockerCli, err := opts.setInstallerContext(dockerCli)
	if err != nil {
		return nil, nil, err
	}
	bind, err := cnab.RequiredClaimBindMount(installation.Claim, dockerCli)
	if err != nil {
		return nil, nil, err
	}
	driverImpl, errBuf := cnab.PrepareDriver(dockerCli, bind, nil)
	return driverImpl, errBuf, nil
}

type parametersOptions struct {
	parametersFiles []string
	overrides       []string
}

func (o *parametersOptions) addFlags(flags *pflag.FlagSet) {
	flags.StringArrayVar(&o.parametersFiles, "parameters-file", []string{}, "Override parameters file")
	flags.StringArrayVarP(&o.overrides, "set", "s", []string{}, "Override parameter value")
}

type installerContextOptions struct {
	installerContext string
}

func (o *installerContextOptions) addFlag(flags *pflag.FlagSet) {
	flags.StringVar(&o.installerContext, "installer-context", "", "Context on which the installer image is ran (default: <current-context>)")
}

func (o *installerContextOptions) setInstallerContext(dockerCli command.Cli) (command.Cli, error) {
	o.installerContext = getTargetContext(o.installerContext, dockerCli.CurrentContext())
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

func getTargetContext(optstargetContext, currentContext string) string {
	var targetContext string
	switch {
	case optstargetContext != "":
		targetContext = optstargetContext
	case os.Getenv("INSTALLER_TARGET_CONTEXT") != "":
		targetContext = os.Getenv("INSTALLER_TARGET_CONTEXT")
	}
	if targetContext == "" {
		targetContext = currentContext
	}
	return targetContext
}

type credentialOptions struct {
	credentialsets   []string
	credentials      []string
	sendRegistryAuth bool
}

func (o *credentialOptions) addFlags(flags *pflag.FlagSet) {
	flags.StringArrayVar(&o.credentialsets, "credential-set", []string{}, "Use a YAML file containing a credential set or a credential set present in the credential store")
	flags.StringArrayVar(&o.credentials, "credential", nil, "Add a single credential, additive ontop of any --credential-set used")
	flags.BoolVar(&o.sendRegistryAuth, "with-registry-auth", false, "Sends registry auth")
}

func (o *credentialOptions) CredentialSetOpts(dockerCli command.Cli, credentialStore store.CredentialStore) []credentialSetOpt {
	return []credentialSetOpt{
		addNamedCredentialSets(credentialStore, o.credentialsets),
		addCredentials(o.credentials),
		addDockerCredentials(dockerCli.CurrentContext(), dockerCli.ContextStore()),
		addRegistryCredentials(o.sendRegistryAuth, dockerCli),
	}
}

func isInstallationFailed(installation *appstore.Installation) bool {
	return installation.Result.Action == claim.ActionInstall &&
		installation.Result.Status == claim.StatusFailure
}
