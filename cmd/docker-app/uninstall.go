package main

import (
	"fmt"

	"github.com/deislabs/duffle/pkg/action"
	"github.com/deislabs/duffle/pkg/claim"
	"github.com/deislabs/duffle/pkg/credentials"
	"github.com/deislabs/duffle/pkg/utils/crud"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/spf13/cobra"
)

type uninstallOptions struct {
	targetContext  string
	credentialsets []string
}

func uninstallCmd(dockerCli command.Cli) *cobra.Command {
	var opts uninstallOptions

	cmd := &cobra.Command{
		Use:   "uninstall <installation-name>",
		Short: "Uninstall an application",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUninstall(dockerCli, args[0], opts)
		},
	}

	cmd.Flags().StringVar(&opts.targetContext, "target-context", "", "Context on which to uninstall the application")
	cmd.Flags().StringArrayVarP(&opts.credentialsets, "credential-set", "c", []string{}, "Use a duffle credentialset (either a YAML file, or a credential set present in the duffle credential store)")

	return cmd
}

func runUninstall(dockerCli command.Cli, claimName string, opts uninstallOptions) error {
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
	uninst := &action.Uninstall{
		Driver: driverImpl,
	}
	err = uninst.Run(&c, creds, dockerCli.Out())
	if err == nil {
		return claimStore.Delete(claimName)
	}
	if err2 := claimStore.Store(c); err2 != nil {
		fmt.Fprintf(dockerCli.Err(), "failed to update claim: %s\n", err2)
	}
	return err
}
