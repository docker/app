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

type uninstallOptions struct {
	credentialOptions
	force bool
}

func uninstallCmd(dockerCli command.Cli) *cobra.Command {
	var opts uninstallOptions

	cmd := &cobra.Command{
		Use:     "uninstall INSTALLATION_NAME [--target-context TARGET_CONTEXT] [OPTIONS]",
		Short:   "Uninstall an application",
		Example: `$ docker app uninstall myinstallation --target-context=mycontext`,
		Args:    cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUninstall(dockerCli, args[0], opts)
		},
	}
	opts.addFlags(cmd.Flags())
	cmd.Flags().BoolVar(&opts.force, "force", false, "Force removal of installation")

	return cmd
}

func runUninstall(dockerCli command.Cli, installationName string, opts uninstallOptions) (mainErr error) {
	defer muteDockerCli(dockerCli)()
	opts.SetDefaultTargetContext(dockerCli)

	_, installationStore, credentialStore, err := prepareStores(opts.targetContext)
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
				fmt.Fprintf(os.Stderr, "failed to force deletion of installation %q: %s\n", installationName, err)
				return
			}
			fmt.Fprintf(os.Stderr, "deletion forced for installation %q\n", installationName)
		}()
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
	uninst := &action.Uninstall{
		Driver: driverImpl,
	}
	if err := uninst.Run(&installation.Claim, creds, os.Stdout); err != nil {
		if err2 := installationStore.Store(installation); err2 != nil {
			return fmt.Errorf("%s", err2)
		}
		return fmt.Errorf("Uninstall failed: %s\n%s", err, errBuf)
	}
	if err := installationStore.Delete(installationName); err != nil {
		return fmt.Errorf("Failed to delete installation %q from the installation store: %s", installationName, err)
	}
	fmt.Fprintf(os.Stdout, "Application %q uninstalled on context %q\n", installationName, opts.targetContext)
	return nil
}
