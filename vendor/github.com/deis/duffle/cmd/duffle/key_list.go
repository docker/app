package main

import (
	"fmt"
	"io"

	"github.com/deis/duffle/pkg/duffle/home"
	"github.com/deis/duffle/pkg/signature"

	"github.com/gosuri/uitable"
	"github.com/spf13/cobra"
)

const keyListDesc = `List key IDs for both public (verify-only) and private (sign and verify) keys.

By default, this lists both signing and verifying keys. All signing keys can be used
to verify. But a verify-only key can not be used to sign.

Use the '--signing' flag to list just the signing keys, and the '--verify-only' flag to
list just the public keys, which can only be used for verifying.

Because a key can exist in both the public and the secret keyring, it is possible for a
key to show up as a signing key with '--signing', and a verifying key with '--verify-only'.
This simply means that the key has been added to both keyrings.
`

func newKeyListCmd(w io.Writer) *cobra.Command {
	var (
		privateOnly bool
		publicOnly  bool
		long        bool
	)
	cmd := &cobra.Command{
		Use:   "list",
		Short: "list key IDs",
		Long:  keyListDesc,
		RunE: func(cmd *cobra.Command, args []string) error {
			h := home.Home(homePath())
			// Order is important, since duplicate keys are skipped, and a key
			// can appear in both keyrings.
			rings := []string{h.SecretKeyRing(), h.PublicKeyRing()}
			if privateOnly {
				rings = []string{h.SecretKeyRing()}
			}
			if publicOnly {
				rings = []string{h.PublicKeyRing()}
			}
			return listKeys(cmd.OutOrStdout(), long, rings...)
		},
	}
	cmd.Flags().BoolVarP(&privateOnly, "signing", "s", false, "show private (sign-or-verify) keys")
	cmd.Flags().BoolVarP(&publicOnly, "verify-only", "p", false, "show public (verify-only) keys")
	cmd.Flags().BoolVarP(&long, "long", "l", false, "show additional details")

	return cmd
}

func listKeys(out io.Writer, long bool, rings ...string) error {
	kr, err := signature.LoadKeyRings(rings...)
	if err != nil {
		return err
	}

	if !long {
		for _, k := range kr.Keys() {
			name, err := k.UserID()
			if err != nil {
				fmt.Fprintln(out, "[anonymous key]")
			}
			fmt.Fprintln(out, name)
		}
		return nil
	}
	table := uitable.New()
	table.MaxColWidth = 80
	table.Wrap = true

	table.AddRow("NAME", "TYPE", "FINGERPRINT")
	for _, k := range kr.Keys() {
		var name, fingerprint string
		id, err := k.UserID()
		if err != nil {
			name = "[anonymous key]"
		} else {
			name = id.String()
		}
		fingerprint = k.Fingerprint()
		typ := "verify-only"
		if k.CanSign() {
			typ = "signing"
		}
		table.AddRow(name, typ, fingerprint)
	}
	fmt.Fprintln(out, table)
	return nil
}
