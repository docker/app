package main

import (
	"fmt"

	"github.com/deislabs/duffle/pkg/action"
	"github.com/deislabs/duffle/pkg/claim"
	"github.com/deislabs/duffle/pkg/credentials"
	"github.com/deislabs/duffle/pkg/utils/crud"
	"github.com/docker/cli/cli/command"
	"github.com/spf13/cobra"
)

type upgradeOptions struct {
	parametersOptions
	credentialOptions
	bundleOrDockerApp string
	insecure          bool
}

func upgradeCmd(dockerCli command.Cli) *cobra.Command {
	var opts upgradeOptions
	cmd := &cobra.Command{
		Use:   "upgrade <installation-name> [options]",
		Short: "Upgrade an installed application",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpgrade(dockerCli, args[0], opts)
		},
	}
	opts.parametersOptions.addFlags(cmd.Flags())
	opts.credentialOptions.addFlags(cmd.Flags())
	cmd.Flags().StringVar(&opts.bundleOrDockerApp, "bundle", "", "Override with new bundle or Docker App")
	cmd.Flags().BoolVar(&opts.insecure, "insecure", false, "Use insecure registry, without SSL")

	return cmd
}

func runUpgrade(dockerCli command.Cli, installationName string, opts upgradeOptions) error {
	defer muteDockerCli(dockerCli)()
	targetContext := getTargetContext(opts.targetContext, dockerCli.CurrentContext())
	h := duffleHome()
	claimStore := claim.NewClaimStore(crud.NewFileSystemStore(h.Claims(), "json"))
	c, err := claimStore.Read(installationName)
	if err != nil {
		return err
	}

	if opts.bundleOrDockerApp != "" {
		b, err := resolveBundle(dockerCli, opts.bundleOrDockerApp)
		if err != nil {
			return err
		}
		c.Bundle = b
	}
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

	c.Parameters, err = mergeBundleParameters(c.Bundle,
		withFileParameters(opts.parametersFiles),
		withCommandLineParameters(opts.overrides),
	)
	if err != nil {
		return err
	}

	u := &action.Upgrade{
		Driver: driverImpl,
	}
	err = u.Run(&c, creds, dockerCli.Out())
	err2 := claimStore.Store(c)
	if err != nil {
		return fmt.Errorf("upgrade failed: %v", err)
	}
	return err2
}
