package commands

import (
	"fmt"
	"os"

	"github.com/deislabs/cnab-go/action"
	"github.com/deislabs/cnab-go/credentials"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/spf13/cobra"
)

type removeOptions struct {
	credentialOptions
	installerContextOptions
	force bool
}

func removeCmd(dockerCli command.Cli) *cobra.Command {
	var opts removeOptions

	cmd := &cobra.Command{
		Use:     "rm [OPTIONS] RUNNING_APP",
		Short:   "Remove a running App",
		Aliases: []string{"remove"},
		Example: `$ docker app rm myrunningapp`,
		Args:    cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRemove(dockerCli, args[0], opts)
		},
	}
	opts.credentialOptions.addFlags(cmd.Flags())
	opts.installerContextOptions.addFlags(cmd.Flags())
	cmd.Flags().BoolVar(&opts.force, "force", false, "Force the removal of a running App")

	return cmd
}

func runRemove(dockerCli command.Cli, installationName string, opts removeOptions) (mainErr error) {
	defer muteDockerCli(dockerCli)()

	_, installationStore, credentialStore, err := prepareStores(dockerCli.CurrentContext())
	if err != nil {
		return err
	}

	installation, err := installationStore.Read(installationName)
	if err != nil {
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
	uninst := &action.Uninstall{
		Driver: driverImpl,
	}
	if err := uninst.Run(&installation.Claim, creds, os.Stdout); err != nil {
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
