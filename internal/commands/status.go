package commands

import (
	"fmt"

	"github.com/deislabs/duffle/pkg/action"
	"github.com/deislabs/duffle/pkg/credentials"
	"github.com/docker/app/internal"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/spf13/cobra"
)

func statusCmd(dockerCli command.Cli) *cobra.Command {
	var opts credentialOptions

	cmd := &cobra.Command{
		Use:     "status INSTALLATION_NAME [--target-context TARGET_CONTEXT] [OPTIONS]",
		Short:   "Get the installation status of an application",
		Long:    "Get the installation status of an application. If the installation is a Docker Application, the status shows the stack services.",
		Example: "$ docker app status myinstallation --target-context=mycontext",
		Args:    cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatus(dockerCli, args[0], opts)
		},
	}
	opts.addFlags(cmd.Flags())

	return cmd
}

func runStatus(dockerCli command.Cli, installationName string, opts credentialOptions) error {
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
	bind, err := requiredClaimBindMount(installation.Claim, opts.targetContext, dockerCli)
	if err != nil {
		return err
	}
	driverImpl, errBuf, err := prepareDriver(dockerCli, bind, nil)
	if err != nil {
		return err
	}
	if err := mergeBundleParameters(installation,
		withSendRegistryAuth(opts.sendRegistryAuth),
	); err != nil {
		return err
	}
	creds, err := prepareCredentialSet(installation.Bundle, opts.CredentialSetOpts(dockerCli, credentialStore)...)
	if err != nil {
		return err
	}
	if err := credentials.Validate(creds, installation.Bundle.Credentials); err != nil {
		return err
	}
	status := &action.RunCustom{
		Action: internal.ActionStatusName,
		Driver: driverImpl,
	}
	if err := status.Run(&installation.Claim, creds, dockerCli.Out()); err != nil {
		return fmt.Errorf("status failed: %s\n%s", err, errBuf)
	}
	return nil
}
