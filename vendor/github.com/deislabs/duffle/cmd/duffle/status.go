package main

import (
	"errors"
	"fmt"
	"io"

	"github.com/gosuri/uitable"
	"github.com/spf13/cobra"

	"github.com/deislabs/duffle/pkg/action"
	"github.com/deislabs/duffle/pkg/claim"
)

func newStatusCmd(w io.Writer) *cobra.Command {
	const short = "get the status of an installation"
	const long = `Gets the status of an existing installation.

Given an installation name, execute the status task for this. A status
action will restart the CNAB image and ask it to query for status. For that
reason, it may need the same credentials used to install.
`
	var (
		statusDriver     string
		credentialsFiles []string
	)

	cmd := &cobra.Command{
		Use:   "status NAME",
		Short: short,
		Long:  long,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return errors.New("required arg is NAME (installation name")
			}
			claimName := args[0]
			c, err := loadClaim(claimName)
			if err != nil {
				if err == claim.ErrClaimNotFound {
					return fmt.Errorf("Bundle installation '%s' not found", claimName)
				}
				return err
			}

			//display information about the bundle installation
			table := uitable.New()
			table.MaxColWidth = 80
			table.Wrap = true

			table.AddRow("Installation Name:", c.Name)
			table.AddRow("Installed at:", c.Created)
			table.AddRow("Last Modified at:", c.Modified)
			table.AddRow("Current Revision:", c.Revision)
			table.AddRow("Bundle:", c.Bundle.Name)
			table.AddRow("Last Action Performed:", c.Result.Action)
			table.AddRow("Last Action Status:", c.Result.Status)
			table.AddRow("Last Action Message:", c.Result.Message)
			fmt.Println(table)

			creds, err := loadCredentials(credentialsFiles, c.Bundle)
			if err != nil {
				return err
			}

			driverImpl, err := prepareDriver(statusDriver)
			if err != nil {
				return err
			}

			// TODO: Do we pass new values in here? Or just from Claim?
			action := &action.Status{Driver: driverImpl}
			fmt.Println("Executing status action in bundle...")
			return action.Run(&c, creds, cmd.OutOrStdout())
		},
	}
	cmd.Flags().StringVarP(&statusDriver, "driver", "d", "docker", "Specify a driver name")
	cmd.Flags().StringArrayVarP(&credentialsFiles, "credentials", "c", []string{}, "Specify credentials to use inside the CNAB bundle. This can be a credentialset name or a path to a file.")

	return cmd
}

func loadClaim(name string) (claim.Claim, error) {
	storage := claimStorage()
	return storage.Read(name)
}
