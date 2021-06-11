package commands

import (
	"fmt"
	"os"

	"github.com/docker/app/internal/cliopts"
	"github.com/docker/app/internal/cnab"
	"github.com/docker/app/internal/packager"
	"github.com/docker/app/internal/store"

	"github.com/cnabio/cnab-go/driver"

	"github.com/cnabio/cnab-go/action"
	"github.com/cnabio/cnab-go/credentials"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	multierror "github.com/hashicorp/go-multierror"
	"github.com/spf13/cobra"
)

type removeOptions struct {
	credentialOptions
	force bool
}

func removeCmd(dockerCli command.Cli, installerContext *cliopts.InstallerContextOptions) *cobra.Command {
	var opts removeOptions

	cmd := &cobra.Command{
		Use:     "rm [OPTIONS] RUNNING_APP",
		Short:   "Remove a running App",
		Aliases: []string{"remove"},
		Example: `$ docker app rm myrunningapp`,
		Args:    cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, installationStore, credentialStore, err := prepareStores(dockerCli.CurrentContext())
			if err != nil {
				return err
			}

			var failures *multierror.Error
			for _, arg := range args {
				if err := runRemove(dockerCli, arg, opts, installerContext, installationStore, credentialStore); err != nil {
					failures = multierror.Append(failures, err)
				}
			}
			return failures.ErrorOrNil()
		},
	}
	opts.credentialOptions.addFlags(cmd.Flags())
	cmd.Flags().BoolVarP(&opts.force, "force", "f", false, "Force the removal of a running App")

	return cmd
}

func runRemove(dockerCli command.Cli,
	installationName string,
	opts removeOptions,
	installerContext *cliopts.InstallerContextOptions,
	installationStore store.InstallationStore,
	credentialStore store.CredentialStore) (mainErr error) {
	installation, err := installationStore.Read(installationName)
	if err != nil {
		return err
	}
	if err := packager.CheckAppVersion(dockerCli.Err(), installation.Bundle); err != nil {
		return err
	}

	if opts.force {
		defer func() {
			if mainErr == nil {
				return
			}
			if err := installationStore.Delete(installationName); err != nil {
				fmt.Fprintf(os.Stderr, "failed to force deletion of running App %q: %s\n", installationName, err)
				return
			}
			fmt.Fprintf(os.Stderr, "deletion forced for running App %q\n", installationName)
		}()
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
	uninst := &action.Uninstall{
		Driver: driverImpl,
	}
	cfgFunc := func(op *driver.Operation) error {
		op.Out = dockerCli.Out()
		return nil
	}
	if err := uninst.Run(&installation.Claim, creds, cfgFunc, cnab.WithRelocationMap(installation)); err != nil {
		if err2 := installationStore.Store(installation); err2 != nil {
			return fmt.Errorf("%s while %s", err2, errBuf)
		}
		return fmt.Errorf("Remove failed: %s\n%s", err, errBuf)
	}
	if err := installationStore.Delete(installationName); err != nil {
		return fmt.Errorf("Failed to delete running App %q from the installation store: %s", installationName, err)
	}
	fmt.Fprintf(dockerCli.Out(), "App %q uninstalled on context %q\n", installationName, dockerCli.CurrentContext())
	return nil
}
