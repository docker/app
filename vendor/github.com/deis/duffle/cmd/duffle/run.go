package main

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/deis/duffle/pkg/action"
	"github.com/deis/duffle/pkg/claim"
)

func newRunCmd(w io.Writer) *cobra.Command {
	const short = "run a target in the bundle"
	const long = `Run an arbitrary target in the bundle.

Some CNAB bundles may declare custom targets in addition to install, upgrade, and uninstall.
This command can be used to execute those targets.

The 'run' command takes a ACTION and a RELEASE NAME:

  $ duffle run migrate my-release

This will start the invocation image for the release in 'my-release', and then send
the action 'migrate'. If the invocation image does not have a 'migrate' action, it
may return an error.

Custom actions can only be executed on releases (already-installed bundles).

Credentials and parameters may be passed to the bundle during a target action.
`
	var (
		driver           string
		credentialsFiles []string
		valuesFile       string
		setParams        []string
		setFiles         []string
	)

	cmd := &cobra.Command{
		Use:     "run ACTION RELEASE_NAME",
		Aliases: []string{"exec"},
		Short:   short,
		Long:    long,
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := args[0]
			claimName := args[1]
			storage := claimStorage()
			c, err := storage.Read(claimName)
			if err != nil {
				if err == claim.ErrClaimNotFound {
					return fmt.Errorf("Bundle installation '%s' not found", claimName)
				}
				return err
			}

			creds, err := loadCredentials(credentialsFiles, c.Bundle)
			if err != nil {
				return err
			}

			driverImpl, err := prepareDriver(driver)
			if err != nil {
				return err
			}

			// Override parameters only if some are set.
			if valuesFile != "" || len(setParams) > 0 {
				c.Parameters, err = calculateParamValues(c.Bundle, valuesFile, setParams, setFiles)
				if err != nil {
					return err
				}
			}

			action := &action.RunCustom{
				Driver: driverImpl,
				Action: target,
			}

			fmt.Printf("Executing custom action %q for release %q", target, claimName)
			err = action.Run(&c, creds, cmd.OutOrStdout())
			if actionDef := c.Bundle.Actions[target]; !actionDef.Modifies {
				// Do not store a claim for non-mutating actions.
				return err
			}

			err2 := storage.Store(c)
			if err != nil {
				return fmt.Errorf("run failed: %s", err)
			}
			return err2
		},
	}
	flags := cmd.Flags()
	flags.StringVarP(&driver, "driver", "d", "docker", "Specify a driver name")
	flags.StringArrayVarP(&credentialsFiles, "credentials", "c", []string{}, "Specify a set of credentials to use inside the CNAB bundle")
	flags.StringVarP(&valuesFile, "parameters", "p", "", "Specify file containing parameters. Formats: toml, MORE SOON")
	flags.StringArrayVarP(&setParams, "set", "s", []string{}, "Set individual parameters as NAME=VALUE pairs")

	return cmd
}
