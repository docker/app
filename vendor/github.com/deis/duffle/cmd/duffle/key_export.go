package main

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/deis/duffle/pkg/duffle/home"
	"github.com/deis/duffle/pkg/signature"

	"github.com/spf13/cobra"
)

const keyExportDesc = `Export the public key part of a signing key.

This exports a public key to a file (as a keyring). If the output file already
exists, the new key will be added. Importantly, if other private keys exist in
the file, the private key material will be removed from those as well.

If no key name is given, the default signing key is exported.
`

func newKeyExportCmd(w io.Writer) *cobra.Command {
	var dest string
	var keyname string
	cmd := &cobra.Command{
		Use:   "export FILE",
		Short: "export the public key of a signing key",
		Long:  keyExportDesc,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			h := home.Home(homePath())
			dest = args[0]

			ring, err := signature.LoadKeyRing(h.SecretKeyRing())
			if err != nil {
				return err
			}

			// Get either the named key or the first key in the ring
			var key *signature.Key
			if keyname == "" {
				allKeys := ring.Keys()
				if len(allKeys) == 0 {
					return errors.New("no keys found in signing ring")
				}
				key = allKeys[0]
			} else {
				key, err = ring.Key(keyname)
				if err != nil {
					return err
				}
			}

			uid, err := key.UserID()
			if err != nil {
				return fmt.Errorf("could not load key with ID %q: %s", keyname, err)
			}
			if verbose {
				fmt.Fprintf(w, "found key %q matching %q\n", uid.String(), keyname)
			}

			// Send to the destination, or to STDOUT
			if fi, err := os.Stat(dest); os.IsNotExist(err) {
				kr := signature.CreateKeyRing(passwordFetcher)
				kr.AddKey(key)
				return kr.SavePublic(dest, true)
			} else if err != nil {
				return err
			} else if fi.IsDir() {
				return errors.New("destination cannot be a directory")
			}

			kr, err := signature.LoadKeyRing(dest)
			if err != nil {
				return err
			}
			kr.AddKey(key)

			for _, k := range kr.Keys() {
				uid, _ := k.UserID()
				println(uid.String())
			}
			return kr.SavePublic(dest, true)

		},
	}
	cmd.Flags().StringVarP(&keyname, "user", "u", "", "the user ID of the key to export")
	return cmd
}
