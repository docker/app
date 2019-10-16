package commands

import (
	"fmt"
	"os"

	"github.com/deislabs/cnab-go/action"
	"github.com/deislabs/cnab-go/credentials"
	"github.com/docker/cli/cli/command"
	"github.com/spf13/cobra"
)

type updateOptions struct {
	parametersOptions
	credentialOptions
	bundleOrDockerApp string
}

func updateCmd(dockerCli command.Cli) *cobra.Command {
	var opts updateOptions
	cmd := &cobra.Command{
		Use:     "update [OPTIONS] RUNNING_APP",
		Short:   "Update a running application",
		Example: `$ docker app update myrunningapp --target-context=mycontext --set key=value`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpdate(dockerCli, args[0], opts)
		},
	}
	opts.parametersOptions.addFlags(cmd.Flags())
	opts.credentialOptions.addFlags(cmd.Flags())
	cmd.Flags().StringVar(&opts.bundleOrDockerApp, "image", "", "Override the installation with another Application Package")

	return cmd
}

func runUpdate(dockerCli command.Cli, installationName string, opts updateOptions) error {
	defer muteDockerCli(dockerCli)()
	opts.SetDefaultTargetContext(dockerCli)

	bundleStore, installationStore, credentialStore, err := prepareStores(opts.targetContext)
	if err != nil {
		return err
	}

	installation, err := installationStore.Read(installationName)
	if err != nil {
		return err
	}

	if isInstallationFailed(installation) {
		return fmt.Errorf("Installation %q has failed and cannot be updated, reinstall it using 'docker app install'", installationName)
	}

	if opts.bundleOrDockerApp != "" {
		b, _, err := resolveBundle(dockerCli, bundleStore, opts.bundleOrDockerApp)
		if err != nil {
			return err
		}
		installation.Bundle = b
	}
	if err := mergeBundleParameters(installation,
		withFileParameters(opts.parametersFiles),
		withCommandLineParameters(opts.overrides),
		withSendRegistryAuth(opts.sendRegistryAuth),
	); err != nil {
		return err
	}

	bind, err := requiredClaimBindMount(installation.Claim, opts.targetContext, dockerCli)
	if err != nil {
		return err
	}
	driverImpl, errBuf := prepareDriver(dockerCli, bind, nil)
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
	fmt.Fprintf(os.Stdout, "Application %q updated on context %q\n", installationName, opts.targetContext)
	return nil
}
