package main

import (
	"github.com/deislabs/duffle/pkg/action"
	"github.com/deislabs/duffle/pkg/claim"
	"github.com/deislabs/duffle/pkg/credentials"
	"github.com/deislabs/duffle/pkg/utils/crud"
	"github.com/docker/app/internal"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/pkg/errors"
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

	driverImpl, err := prepareDriver(dockerCli)
	if err != nil {
		return err
	}
	creds, err := prepareCredentialSet(targetContext, dockerCli.ContextStore(), c.Bundle, opts.credentialsets)
	if err != nil {
		return err
	}
	if err := credentials.Validate(creds, c.Bundle.Credentials); err != nil {
		return err
	}
	status := &action.RunCustom{
		Action: internal.Namespace + "status",
		Driver: driverImpl,
	}
	err = status.Run(&c, creds, dockerCli.Out())
	return errors.Wrap(err, "Status failed")
}
