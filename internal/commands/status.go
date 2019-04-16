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
		Use:   "status <installation-name>",
		Short: "Get the installation status. If the installation is a docker application, the status shows the stack services.",
		Args:  cli.ExactArgs(1),
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

	c, err := installationStore.Read(installationName)
	if err != nil {
		return err
	}
	bind, err := requiredClaimBindMount(c, opts.targetContext, dockerCli)
	if err != nil {
		return err
	}
	driverImpl, errBuf, err := prepareDriver(dockerCli, bind, nil)
	if err != nil {
		return err
	}
	if err := mergeBundleParameters(&c,
		withSendRegistryAuth(opts.sendRegistryAuth),
	); err != nil {
		return err
	}
	creds, err := prepareCredentialSet(c.Bundle, opts.CredentialSetOpts(dockerCli, credentialStore)...)
	if err != nil {
		return err
	}
	if err := credentials.Validate(creds, c.Bundle.Credentials); err != nil {
		return err
	}
	status := &action.RunCustom{
		Action: internal.ActionStatusName,
		Driver: driverImpl,
	}
	if err := status.Run(&c, creds, dockerCli.Out()); err != nil {
		return fmt.Errorf("status failed: %s\n%s", err, errBuf)
	}
	return nil
}
