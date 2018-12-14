package main

import (
	"encoding/json"
	"io"

	"github.com/deislabs/duffle/pkg/claim"

	"github.com/spf13/cobra"
)

const claimsShowDesc = `
Display the content of a claim.

This dumps the entire content of a claim as a JSON object.
`

func newClaimsShowCmd(w io.Writer) *cobra.Command {
	var onlyBundle bool
	cmd := &cobra.Command{
		Use:     "show NAME",
		Short:   "show a claim",
		Long:    claimsShowDesc,
		Aliases: []string{"get"},
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			storage := claimStorage()
			return displayClaim(name, w, storage, onlyBundle)
		},
	}

	cmd.Flags().BoolVarP(&onlyBundle, "bundle", "b", false, "only show the bundle from the claim")

	return cmd
}

func displayClaim(name string, out io.Writer, storage claim.Store, onlyBundle bool) error {
	c, err := storage.Read(name)
	if err != nil {
		return err
	}

	var data []byte
	if onlyBundle {
		data, err = json.MarshalIndent(c.Bundle, "", "  ")
	} else {
		data, err = json.MarshalIndent(c, "", "  ")
	}
	if err != nil {
		return err
	}

	_, err = out.Write(data)
	out.Write([]byte("\n"))
	return err
}
