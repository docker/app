package main

import (
	"errors"
	"fmt"

	"github.com/deislabs/duffle/pkg/action"

	"github.com/spf13/cobra"
)

const upgradeUsage = `perform the upgrade action in the CNAB bundle`
const upgradeLong = `Upgrades an existing application.

An upgrade can do the following:

	- Upgrade a current release to a newer bundle (optionally with parameters)
	- Upgrade a current release using the same bundle but different parameters

Credentials must be supplied when applicable, though they need not be the same credentials that were used
to do the install.

If no parameters are passed, the parameters from the previous release will be used. If '--set' or '--parameters'
are specified, the parameters there will be used (even if the resolved set is empty).
`

var upgradeDriver string

type upgradeCmd struct {
	duffleCmd
	name       string
	valuesFile string
	setParams  []string
	insecure   bool
	setFiles   []string
}

func newUpgradeCmd() *cobra.Command {
	uc := &upgradeCmd{}

	var (
		credentialsFiles []string
		bundleFile       string
	)

	cmd := &cobra.Command{
		Use:     "upgrade NAME [BUNDLE]",
		Short:   upgradeUsage,
		Long:    upgradeLong,
		PreRunE: uc.Prepare(),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return errors.New("This command requires at least 1 argument: the name of the installation to upgrade")
			}
			uc.name = args[0]
			uc.Out = cmd.OutOrStdout()
			bundleFile, err := optBundleFileOrArg2(args, bundleFile, uc.Out, uc.insecure)
			if err != nil {
				return err
			}

			return uc.upgrade(credentialsFiles, bundleFile)
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&bundleFile, "file", "f", "", "Set the bundle file to use for upgrading")
	flags.StringVarP(&upgradeDriver, "driver", "d", "docker", "Specify a driver name")
	flags.StringArrayVarP(&credentialsFiles, "credentials", "c", []string{}, "Specify credentials to use inside the CNAB bundle. This can be a credentialset name or a path to a file.")
	flags.StringVarP(&uc.valuesFile, "parameters", "p", "", "Specify file containing parameters. Formats: toml, MORE SOON")
	flags.StringArrayVarP(&uc.setParams, "set", "s", []string{}, "Set individual parameters as NAME=VALUE pairs")
	flags.BoolVarP(&uc.insecure, "insecure", "k", false, "Do not verify the bundle (INSECURE)")
	flags.StringArrayVarP(&uc.setFiles, "set-file", "i", []string{}, "Set individual parameters from file content as NAME=SOURCE-PATH pairs")
	return cmd
}

func (up *upgradeCmd) upgrade(credentialsFiles []string, bundleFile string) error {

	claim, err := claimStorage().Read(up.name)
	if err != nil {
		return fmt.Errorf("%v not found: %v", up.name, err)
	}

	// If the user specifies a bundle file, override the existing one.
	if bundleFile != "" {
		bun, err := loadBundle(bundleFile, up.insecure)
		if err != nil {
			return err
		}
		claim.Bundle = bun
	}

	driverImpl, err := prepareDriver(upgradeDriver)
	if err != nil {
		return err
	}

	creds, err := loadCredentials(credentialsFiles, claim.Bundle)
	if err != nil {
		return err
	}

	// Override parameters only if some are set.
	if up.valuesFile != "" || len(up.setParams) > 0 {
		claim.Parameters, err = calculateParamValues(claim.Bundle, up.valuesFile, up.setParams, up.setFiles)
		if err != nil {
			return err
		}
	}

	upgr := &action.Upgrade{
		Driver: driverImpl,
	}
	err = upgr.Run(&claim, creds, up.Out)

	// persist the claim, regardless of the success of the upgrade action
	persistErr := claimStorage().Store(claim)

	if err != nil {
		return fmt.Errorf("could not upgrade %q: %s", up.name, err)
	}
	return persistErr
}
