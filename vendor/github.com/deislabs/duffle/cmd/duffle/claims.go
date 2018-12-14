package main

import (
	"io"

	"github.com/spf13/cobra"
)

const claimsDesc = `
Work with claims and existing releases.

A claim is a record of a release. When a bundle is installed, Duffle retains a
claim that tracks that release. Subsequent operations (like upgrades) will
modify the claim record.

The claim tools provide features for working directly with claims.
`

func newClaimsCmd(w io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "claims",
		Short:   "manage claims",
		Long:    claimsDesc,
		Aliases: []string{"claim"},
	}

	cmd.AddCommand(newClaimsShowCmd(w))
	cmd.AddCommand(newClaimListCmd(w))

	return cmd
}
