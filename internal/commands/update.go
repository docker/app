package commands

import (
	"fmt"
	"os"

	"github.com/deislabs/cnab-go/driver"

	"github.com/deislabs/cnab-go/action"
	"github.com/deislabs/cnab-go/credentials"
	"github.com/docker/app/internal/bundle"
	"github.com/docker/app/internal/cliopts"
	"github.com/docker/app/internal/cnab"
	"github.com/docker/app/internal/packager"
	"github.com/docker/cli/cli/command"
	"github.com/spf13/cobra"
)

type updateOptions struct {
	cliopts.ParametersOptions
	credentialOptions
	bundleOrDockerApp string
}

func updateCmd(dockerCli command.Cli, installerContext *cliopts.InstallerContextOptions) *cobra.Command {
	var opts updateOptions
	cmd := &cobra.Command{
		Use:     "update [OPTIONS] RUNNING_APP",
		Short:   "Update a running App",
		Example: `$ docker app update myrunningapp --set key=value`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpdate(dockerCli, args[0], opts, installerContext)
		},
	}
	opts.ParametersOptions.AddFlags(cmd.Flags())
	opts.credentialOptions.addFlags(cmd.Flags())
	cmd.Flags().StringVar(&opts.bundleOrDockerApp, "image", "", "Override the running App with another App image")

	return cmd
}

func runUpdate(dockerCli command.Cli, installationName string, opts updateOptions, installerContext *cliopts.InstallerContextOptions) error {
	bundleStore, installationStore, credentialStore, err := prepareStores(dockerCli.CurrentContext())
	if err != nil {
		return err
	}

	installation, err := installationStore.Read(installationName)
	if err != nil {
		return err
	}

	if IsInstallationFailed(installation) {
		return fmt.Errorf("Running App %q cannot be updated, please use 'docker app run' instead", installationName)
	}

	if opts.bundleOrDockerApp != "" {
		b, _, err := cnab.ResolveBundle(dockerCli, bundleStore, opts.bundleOrDockerApp)
		if err != nil {
			return err
		}
		installation.Bundle = b.Bundle
	}
	if err := packager.CheckAppVersion(dockerCli.Err(), installation.Bundle); err != nil {
		return err
	}

	if err := bundle.MergeBundleParameters(installation,
		bundle.WithFileParameters(opts.ParametersFiles),
		bundle.WithCommandLineParameters(opts.Overrides),
		bundle.WithSendRegistryAuth(opts.sendRegistryAuth),
	); err != nil {
		return err
	}

	defer muteDockerCli(dockerCli)()
	driverImpl, errBuf, err := cnab.SetupDriver(installation, dockerCli, installerContext, os.Stdout)
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
	cfgFunc := func(op *driver.Operation) error {
		op.Out = dockerCli.Out()
		return nil
	}
	err = u.Run(&installation.Claim, creds, cfgFunc, cnab.WithRelocationMap(installation))
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
