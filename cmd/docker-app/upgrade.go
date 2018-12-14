package main

import (
	"fmt"

	"github.com/deislabs/duffle/pkg/action"
	"github.com/deislabs/duffle/pkg/bundle"
	"github.com/deislabs/duffle/pkg/claim"
	"github.com/deislabs/duffle/pkg/credentials"
	"github.com/deislabs/duffle/pkg/utils/crud"
	"github.com/docker/cli/cli/command"
	"github.com/spf13/cobra"
)

type upgradeOptions struct {
	parametersOptions
	targetContext     string
	credentialsets    []string
	bundleOrDockerApp string
	namespace         string
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
	cmd.Flags().StringVar(&opts.targetContext, "target-context", "", "Context on which to upgrade the application")
	cmd.Flags().StringArrayVarP(&opts.credentialsets, "credential-set", "c", []string{}, "Use a duffle credentialset (either a YAML file, or a credential set present in the duffle credential store)")
	cmd.Flags().StringVar(&opts.bundleOrDockerApp, "bundle", "", "Override with new bundle or Docker App")
	cmd.Flags().StringVar(&opts.namespace, "namespace", "", "Namespace to use (default: namespace in metadata)")
	cmd.Flags().BoolVar(&opts.insecure, "insecure", false, "Use insecure registry, without SSL")

	return cmd
}

func runUpgrade(dockerCli command.Cli, installationName string, opts upgradeOptions) error {
	muteDockerCli(dockerCli)
	targetContext := getTargetContext(opts.targetContext)
	parameterValues, err := prepareParameters(opts.parametersOptions)
	if err != nil {
		return err
	}
	h := duffleHome()
	claimStore := claim.NewClaimStore(crud.NewFileSystemStore(h.Claims(), "json"))
	c, err := claimStore.Read(installationName)
	if err != nil {
		return err
	}

	if opts.bundleOrDockerApp != "" {
		b, err := resolveBundle(dockerCli, opts.namespace, opts.bundleOrDockerApp, opts.insecure)
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
	convertedParamValues := c.Parameters
	if err := applyParameterValues(parameterValues, c.Bundle.Parameters, convertedParamValues); err != nil {
		return err
	}

	c.Parameters, err = bundle.ValuesOrDefaults(convertedParamValues, c.Bundle)
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
