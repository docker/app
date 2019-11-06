package commands

import (
	"fmt"
	"os"

	"github.com/deislabs/cnab-go/action"
	"github.com/deislabs/cnab-go/credentials"
	"github.com/docker/app/internal/bundle"
	"github.com/docker/app/internal/cliopts"
	"github.com/docker/app/internal/cnab"
	"github.com/docker/cli/cli/command"
	"github.com/spf13/cobra"
)

type updateOptions struct {
	cliopts.ParametersOptions
	credentialOptions
	installerContextOptions
	bundleOrDockerApp string
}

func updateCmd(dockerCli command.Cli) *cobra.Command {
	var opts updateOptions
	cmd := &cobra.Command{
		Use:     "update [OPTIONS] RUNNING_APP",
		Short:   "Update a running App",
		Example: `$ docker app update myrunningapp --set key=value`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpdate(dockerCli, args[0], opts)
		},
	}
	opts.ParametersOptions.AddFlags(cmd.Flags())
	opts.credentialOptions.addFlags(cmd.Flags())
	opts.installerContextOptions.addFlags(cmd.Flags())
	cmd.Flags().StringVar(&opts.bundleOrDockerApp, "image", "", "Override the running App with another App image")

	return cmd
}

func runUpdate(dockerCli command.Cli, installationName string, opts updateOptions) error {
	defer muteDockerCli(dockerCli)()

	bundleStore, installationStore, credentialStore, err := prepareStores(dockerCli.CurrentContext())
	if err != nil {
		return err
	}

	installation, err := installationStore.Read(installationName)
	if err != nil {
		return err
	}

	if isInstallationFailed(installation) {
		return fmt.Errorf("Running App %q cannot be updated, please use 'docker app run' instead", installationName)
	}

	if opts.bundleOrDockerApp != "" {
		b, _, err := cnab.ResolveBundle(dockerCli, bundleStore, opts.bundleOrDockerApp)
		if err != nil {
			return err
		}
		installation.Bundle = b
	}
	if err := bundle.MergeBundleParameters(installation,
		bundle.WithFileParameters(opts.ParametersFiles),
		bundle.WithCommandLineParameters(opts.Overrides),
		bundle.WithSendRegistryAuth(opts.sendRegistryAuth),
	); err != nil {
		return err
	}

	driverImpl, errBuf, err := setupDriver(installation, dockerCli, opts.installerContextOptions, os.Stdout)
	if err != nil {
		return err
	}
	creds, err := prepareCredentialSet(installation.Bundle, opts.CredentialSetOpts(dockerCli, credentialStore)...)
	if err != nil {
		return err
	}
	if err := credentials.Validate(creds, installation.Bundle.Credentials); err != nil {
		return err
	}
	u := &action.Upgrade{
		Driver: driverImpl,
	}
	err = u.Run(&installation.Claim, creds, os.Stdout)
	err2 := installationStore.Store(installation)
	if err != nil {
		return fmt.Errorf("Update failed: %s\n%s", err, errBuf)
	}
	if err2 != nil {
		return err2
	}
	fmt.Fprintf(dockerCli.Out(), "Running App %q updated on context %q\n", installationName, dockerCli.CurrentContext())
	return nil
}
