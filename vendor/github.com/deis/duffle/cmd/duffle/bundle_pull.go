package main

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/flags"
	"github.com/docker/distribution/reference"
	"github.com/spf13/cobra"

	"github.com/deis/duffle/pkg/crypto/digest"
	"github.com/deis/duffle/pkg/duffle/home"
	"github.com/deis/duffle/pkg/image"
)

func newBundlePullCmd(w io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pull BUNDLE",
		Short: "pull a bundle from an image registry",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli := command.NewDockerCli(os.Stdin, os.Stdout, os.Stderr, false, nil)
			if err := cli.Initialize(flags.NewClientOptions()); err != nil {
				return err
			}
			ref, err := reference.ParseNormalizedNamed(args[0])
			if err != nil {
				return err
			}
			tagged, ok := ref.(reference.NamedTagged)
			if !ok {
				return fmt.Errorf("%q is not a tagged reference. Please specify a version", args[0])
			}
			signedBundle, err := image.PullBundle(context.TODO(), cli, false, tagged.String())
			if err != nil {
				return err
			}
			sha, err := digest.OfBuffer(signedBundle)
			if err != nil {
				return fmt.Errorf("cannot compute digest from bundle: %v", err)
			}

			h := home.Home(homePath())
			fpath := filepath.Join(h.Bundles(), sha)
			if err := ioutil.WriteFile(fpath, signedBundle, 0644); err != nil {
				return err
			}

			return recordBundleReference(h, tagged, sha)
		},
	}
	return cmd
}
