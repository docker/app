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

	"github.com/deis/duffle/pkg/duffle/home"
	"github.com/deis/duffle/pkg/image"
	"github.com/deis/duffle/pkg/repo"
)

func newBundlePushCmd(w io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "push BUNDLE",
		Short: "push a bundle to an image registry",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ref, err := reference.ParseNormalizedNamed(args[0])
			if err != nil {
				return err
			}
			tagged, ok := ref.(reference.NamedTagged)
			if !ok {
				return fmt.Errorf("%q is not a tagged reference. Please specify a version", args[0])
			}
			h := home.Home(homePath())
			index, err := repo.LoadIndex(h.Repositories())
			if err != nil {
				return err
			}
			sha, err := index.GetExactly(tagged)
			if err != nil {
				return err
			}
			fpath := filepath.Join(h.Bundles(), sha)
			data, err := ioutil.ReadFile(fpath)
			if err != nil {
				return err
			}
			cli := command.NewDockerCli(os.Stdin, os.Stdout, os.Stderr, false, nil)
			if err := cli.Initialize(flags.NewClientOptions()); err != nil {
				return err
			}
			digest, err := image.PushBundle(context.TODO(), cli, false, data, tagged.String())
			if err != nil {
				return err
			}
			fmt.Println("Digest of manifest is", digest)
			return nil
		},
	}
	return cmd
}
