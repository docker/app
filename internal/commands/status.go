package commands

import (
	"fmt"

	"github.com/deislabs/duffle/pkg/action"
	"github.com/deislabs/duffle/pkg/claim"
	"github.com/deislabs/duffle/pkg/credentials"
	"github.com/deislabs/duffle/pkg/utils/crud"
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

func runStatus(dockerCli command.Cli, claimName string, opts credentialOptions) error {
	defer muteDockerCli(dockerCli)()
	h := duffleHome()

	claimStore := claim.NewClaimStore(crud.NewFileSystemStore(h.Claims(), "json"))
	c, err := claimStore.Read(claimName)
	if err != nil {
		return err
	}
	targetContext := getTargetContext(opts.targetContext, dockerCli.CurrentContext())
	bind, err := requiredClaimBindMount(c, targetContext, dockerCli)
	if err != nil {
		return err
	}
	driverImpl, errBuf, err := prepareDriver(dockerCli, bind, nil)
	if err != nil {
		return err
	}
	creds, err := prepareCredentialSet(c.Bundle,
		addNamedCredentialSets(opts.credentialsets),
		addDockerCredentials(targetContext, dockerCli.ContextStore()),
		addRegistryCredentials(c.Parameters, dockerCli))
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
