package main

import (
	"fmt"
	"io"
	"io/ioutil"

	"github.com/deislabs/duffle/pkg/duffle/home"
	"github.com/deislabs/duffle/pkg/signature"

	"github.com/spf13/cobra"
)

const bundleVerifyDesc = `Verify a signed bundle.

This command verifies the signature by checking it against both the public
and secret keyrings. A bundle is verified if and only if a key exists in the
keyring(s) that can successfully decrypt the signature and verify the hash.
`

func newBundleVerifyCmd(w io.Writer) *cobra.Command {
	var (
		public     bool
		bundleFile string
	)

	cmd := &cobra.Command{
		Use:   "verify BUNDLE",
		Short: "verify the signature on a signed bundle",
		Long:  bundleVerifyDesc,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			h := home.Home(homePath())
			secret := h.SecretKeyRing()
			public := h.PublicKeyRing()

			bundle, err := bundleFileOrArg1(args, bundleFile)
			if err != nil {
				return err
			}

			return verifySig(bundle, public, secret, w)
		},
	}
	cmd.Flags().BoolVarP(&public, "public", "p", false, "show public key IDs instead of private key IDs")
	cmd.Flags().StringVarP(&bundleFile, "file", "f", "", "path to bundle file to sign")

	return cmd
}

func verifySig(filename, public, private string, out io.Writer) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	ring, err := signature.LoadKeyRing(public)
	if err != nil {
		return err
	}

	// We want the private keyring because a user should be able to verify
	// any bundles that they signed, and their signing key is in the
	// private keyring.
	priv, err := signature.LoadKeyRing(private)
	if err != nil {
		return err
	}
	for _, pk := range priv.Keys() {
		ring.AddKey(pk)
	}

	verifier := signature.NewVerifier(ring)
	key, err := verifier.Verify(data)
	if err != nil {
		return fmt.Errorf("verification failed: %s ", err)
	}

	user, err := key.UserID()
	signed := "[anonymous key]"
	if err == nil {
		signed = user.String()
	}

	fmt.Fprintf(out, "Signed by %q (%s)\n", signed, key.Fingerprint())
	return nil
}
