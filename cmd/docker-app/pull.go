package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/deis/duffle/pkg/bundle"
	"github.com/deis/duffle/pkg/crypto/digest"
	"github.com/deis/duffle/pkg/image"
	"github.com/deis/duffle/pkg/loader"
	"github.com/deis/duffle/pkg/repo"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/distribution/reference"
	"github.com/spf13/cobra"
)

type pullOptions struct {
	insecure bool
}

func pullCmd(dockerCli command.Cli) *cobra.Command {
	var opts pullOptions
	cmd := &cobra.Command{
		Use:   "pull <repotag>",
		Short: "Pull an application from a registry",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := pullBundle(dockerCli, args[0], true, opts.insecure)
			return err
		},
	}
	cmd.Flags().BoolVar(&opts.insecure, "insecure", false, "Use insecure registry, without SSL")
	return cmd
}

func pullBundle(dockerCli command.Cli, name string, force, insecure bool) (*bundle.Bundle, error) {
	named, err := reference.ParseNormalizedNamed(name)
	if err != nil {
		return nil, err
	}
	tagged, ok := named.(reference.NamedTagged)
	if !ok {
		return nil, fmt.Errorf("%q is not a tagged image name", name)
	}
	h := duffleHome()
	index, err := repo.LoadIndex(h.Repositories())
	if err != nil {
		return nil, err
	}
	if !force {
		sha, err := index.GetExactly(tagged)
		if err == nil {
			fpath := filepath.Join(h.Bundles(), sha)
			return loader.NewDetectingLoader().Load(fpath)
		}
	}

	signedBundle, err := image.PullBundle(context.TODO(), dockerCli, insecure, name)
	if err != nil {
		return nil, err
	}
	bndl, err := loader.NewDetectingLoader().LoadData(signedBundle)
	if err != nil {
		return nil, err
	}
	sha, err := digest.OfBuffer(signedBundle)
	if err != nil {
		return nil, fmt.Errorf("cannot compute digest from bundle: %v", err)
	}

	fpath := filepath.Join(h.Bundles(), sha)
	if err := ioutil.WriteFile(fpath, signedBundle, 0644); err != nil {
		return nil, err
	}
	index.Add(tagged, sha)

	if err := index.WriteFile(h.Repositories(), 0644); err != nil {
		return nil, fmt.Errorf("could not write to %s: %v", h.Repositories(), err)
	}
	return bndl, nil
}
