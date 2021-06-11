package commands

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/cnabio/cnab-go/claim"
	"github.com/docker/app/internal"
	"github.com/docker/app/internal/cliopts"
	"github.com/docker/app/internal/commands/build"
	"github.com/docker/app/internal/commands/image"
	"github.com/docker/app/internal/store"
	appstore "github.com/docker/app/internal/store"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/config"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type mainOptions struct {
	cliopts.InstallerContextOptions
	showVersion bool
}

// NewRootCmd returns the base root command.
func NewRootCmd(use string, dockerCli command.Cli) *cobra.Command {
	var opts mainOptions
	cmd := &cobra.Command{
		Short:       "Docker App",
		Long:        `A tool to build, share and run a Docker App`,
		Use:         use,
		Annotations: map[string]string{"experimentalCLI": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.showVersion {
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
	addCommands(cmd, dockerCli, &opts.InstallerContextOptions)

	cmd.Flags().BoolVar(&opts.showVersion, "version", false, "Print version information")
	opts.InstallerContextOptions.AddFlags(cmd.Flags())

	return cmd
}

func addCommands(cmd *cobra.Command, dockerCli command.Cli, installerContext *cliopts.InstallerContextOptions) {
	cmd.AddCommand(
		runCmd(dockerCli, installerContext),
		updateCmd(dockerCli, installerContext),
		removeCmd(dockerCli, installerContext),
		listCmd(dockerCli, installerContext),
		initCmd(dockerCli),
		validateCmd(),
		pushCmd(dockerCli),
		pullCmd(dockerCli),
		image.Cmd(dockerCli, installerContext),
		build.Cmd(dockerCli),
		inspectCmd(dockerCli, installerContext),
	)

	if !dockerCli.ClientInfo().HasExperimental {
		removeExperimentalCmdsAndFlags(cmd)
	}
}

func removeExperimentalCmdsAndFlags(cmd *cobra.Command) {
	enabledFlags := []*pflag.Flag{}
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if _, disabled := f.Annotations["experimentalCLI"]; !disabled {
			enabledFlags = append(enabledFlags, f)
		}
	})

	if len(enabledFlags) != cmd.Flags().NFlag() {
		cmd.ResetFlags()
		for _, f := range enabledFlags {
			cmd.Flags().AddFlag(f)
		}
	}

	for _, subcmd := range cmd.Commands() {
		if _, ok := subcmd.Annotations["experimentalCLI"]; ok {
			cmd.RemoveCommand(subcmd)
		} else {
			removeExperimentalCmdsAndFlags(subcmd)
		}
	}
}

func muteDockerCli(dockerCli command.Cli) func() {
	stdout := dockerCli.Out()
	stderr := dockerCli.Err()
	dockerCli.Apply(command.WithCombinedStreams(ioutil.Discard)) //nolint:errcheck // WithCombinedStreams cannot error
	return func() {
		dockerCli.Apply(command.WithOutputStream(stdout), command.WithErrorStream(stderr)) //nolint:errcheck // as above
	}
}

func prepareStores(targetContext string) (store.ImageStore, store.InstallationStore, store.CredentialStore, error) {
	appstore, err := store.NewApplicationStore(config.Dir())
	if err != nil {
		return nil, nil, nil, err
	}
	installationStore, err := appstore.InstallationStore(targetContext)
	if err != nil {
		return nil, nil, nil, err
	}
	imageStore, err := appstore.ImageStore()
	if err != nil {
		return nil, nil, nil, err
	}
	credentialStore, err := appstore.CredentialStore(targetContext)
	if err != nil {
		return nil, nil, nil, err
	}
	return imageStore, installationStore, credentialStore, nil
}

func prepareImageStore() (store.ImageStore, error) {
	appstore, err := store.NewApplicationStore(config.Dir())
	if err != nil {
		return nil, err
	}
	imageStore, err := appstore.ImageStore()
	if err != nil {
		return nil, err
	}
	return imageStore, nil
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

func IsInstallationFailed(installation *appstore.Installation) bool {
	return installation.Result.Action == claim.ActionInstall &&
		installation.Result.Status == claim.StatusFailure
}
