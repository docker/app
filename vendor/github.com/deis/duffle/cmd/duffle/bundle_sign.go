package main

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/flags"
	"github.com/docker/distribution/reference"
	"github.com/spf13/cobra"

	"github.com/deis/duffle/pkg/bundle"
	"github.com/deis/duffle/pkg/crypto/digest"
	"github.com/deis/duffle/pkg/duffle/home"
	"github.com/deis/duffle/pkg/image"
	"github.com/deis/duffle/pkg/signature"
)

const bundleSignDesc = `Clear-sign a bundle.

This remarshals the bundle.json into canonical form, and then clear-signs the JSON.
By default, the signed bundle is written to $DUFFLE_HOME. You can specify an output-file to save to instead using the flag.

If no key name is supplied, this uses the first signing key in the secret keyring.
`

type bundleSignCmd struct {
	out             io.Writer
	home            home.Home
	identity        string
	bundleFile      string
	outfile         string
	skipValidation  bool
	pushLocalImages bool
}

func newBundleSignCmd(w io.Writer) *cobra.Command {
	sign := &bundleSignCmd{out: w}

	cmd := &cobra.Command{
		Use:   "sign BUNDLE",
		Short: "clear-sign a bundle",
		Args:  cobra.MaximumNArgs(1),
		Long:  bundleSignDesc,
		RunE: func(cmd *cobra.Command, args []string) error {
			sign.home = home.Home(homePath())
			secring := sign.home.SecretKeyRing()
			bundle, err := bundleFileOrArg1(args, sign.bundleFile)
			if err != nil {
				return err
			}
			cli := command.NewDockerCli(os.Stdin, os.Stdout, os.Stderr, false, nil)
			if err := cli.Initialize(flags.NewClientOptions()); err != nil {
				return err
			}
			resolver := image.NewResolver(sign.pushLocalImages, cli)
			return sign.signBundle(bundle, secring, resolver)
		},
	}
	cmd.Flags().StringVarP(&sign.identity, "user", "u", "", "the user ID of the key to use. Format is either email address or 'NAME (COMMENT) <EMAIL>'")
	cmd.Flags().StringVarP(&sign.bundleFile, "file", "f", "", "path to bundle file to sign")
	cmd.Flags().StringVarP(&sign.outfile, "output-file", "o", "", "the name of the output file")
	cmd.Flags().BoolVar(&sign.skipValidation, "skip-validate", false, "do not validate the JSON before marshaling it.")
	cmd.Flags().BoolVar(&sign.pushLocalImages, "push-local-images", false, "push docker local-only images to the registry.")

	return cmd
}

func bundleFileOrArg1(args []string, bundle string) (string, error) {
	switch {
	case len(args) == 1 && bundle != "":
		return "", errors.New("please use either -f or specify a BUNDLE, but not both")
	case len(args) == 0 && bundle == "":
		return "", errors.New("please specify a BUNDLE or use -f for a file")
	case len(args) == 1:
		// passing insecure: true, as currently we can only sign an unsinged bundle
		return getBundleFilepath(args[0], true)
	}
	return bundle, nil
}
func (bs *bundleSignCmd) signBundle(bundleFile, keyring string, containerImageResolver bundle.ContainerImageResolver) error {
	// Verify that file exists
	if fi, err := os.Stat(bundleFile); err != nil {
		return fmt.Errorf("cannot find bundle file to sign: %v", err)
	} else if fi.IsDir() {
		return errors.New("cannot sign a directory")
	}

	bdata, err := ioutil.ReadFile(bundleFile)
	if err != nil {
		return err
	}
	b, err := bundle.Unmarshal(bdata)
	if err != nil {
		return err
	}

	if err := b.FixupContainerImages(containerImageResolver); err != nil {
		if ok, image := image.IsErrImageLocalOnly(err); ok {
			fmt.Fprintf(os.Stderr, "Image %q is only available locally. Please push it to the registry\n", image)
		}
		return err
	}

	if !bs.skipValidation {
		if err := b.Validate(); err != nil {
			return err
		}
	}

	// Load keyring
	kr, err := signature.LoadKeyRing(keyring)
	if err != nil {
		return err
	}
	// Find identity
	var k *signature.Key
	if bs.identity != "" {
		k, err = kr.Key(bs.identity)
		if err != nil {
			return err
		}
	} else {
		all := kr.PrivateKeys()
		if len(all) == 0 {
			return errors.New("no private keys found")
		}
		k = all[0]
	}

	// Sign the file
	s := signature.NewSigner(k)
	data, err := s.Clearsign(b)
	if err != nil {
		return err
	}

	data = append(data, '\n')

	digest, err := digest.OfBuffer(data)
	if err != nil {
		return fmt.Errorf("cannot compute digest from bundle: %v", err)
	}

	// if --output-file is provided, write and return
	if bs.outfile != "" {
		if err := ioutil.WriteFile(bs.outfile, data, 0644); err != nil {
			return err
		}
	}

	named, err := reference.ParseNormalizedNamed(b.Name)
	if err != nil {
		return err
	}

	versioned, err := reference.WithTag(named, b.Version)
	if err != nil {
		return err
	}

	if err := ioutil.WriteFile(filepath.Join(bs.home.Bundles(), digest), data, 0644); err != nil {
		return err
	}

	// TODO - write pkg method in bundle that writes file and records the reference
	if err := recordBundleReference(bs.home, versioned, digest); err != nil {
		return err
	}

	userID, err := k.UserID()
	if err != nil {
		return err
	}
	fmt.Fprintf(bs.out, "Signed by %s %s \n", userID.String(), k.Fingerprint())
	return nil
}
