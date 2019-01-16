package main

import (
	"github.com/deislabs/duffle/pkg/action"
	"github.com/deislabs/duffle/pkg/claim"
	"github.com/deislabs/duffle/pkg/credentials"
	"github.com/deislabs/duffle/pkg/utils/crud"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/spf13/cobra"
)

func statusCmd(dockerCli command.Cli) *cobra.Command {
	var opts uninstallOptions

	cmd := &cobra.Command{
		Use:   "status <installation-name>",
		Short: "Get an application status",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatus(dockerCli, args[0], opts)
		},
	}

	cmd.Flags().StringVar(&opts.targetContext, "target-context", "", "Context on which to request the application status")
	cmd.Flags().StringArrayVarP(&opts.credentialsets, "credential-set", "c", []string{}, "Use a duffle credentialset (either a YAML file, or a credential set present in the duffle credential store)")

	return cmd
}

func runStatus(dockerCli command.Cli, claimName string, opts uninstallOptions) error {
	muteDockerCli(dockerCli)
	h := duffleHome()

	claimStore := claim.NewClaimStore(crud.NewFileSystemStore(h.Claims(), "json"))
	c, err := claimStore.Read(claimName)
	if err != nil {
		return err
	}
	targetContext := getTargetContext(opts.targetContext)

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
	status := &action.Status{
		Driver: driverImpl,
	}
	return status.Run(&c, creds, dockerCli.Out())
}
