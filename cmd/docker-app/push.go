package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/deis/duffle/pkg/bundle"
	"github.com/deis/duffle/pkg/image"
	"github.com/deis/duffle/pkg/signature"
	"github.com/docker/app/internal/packager"
	"github.com/docker/app/types"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/spf13/cobra"
)

type pushOptions struct {
	namespace string
	tag       string
	repo      string
	insecure  bool
}

func pushCmd(dockerCli command.Cli) *cobra.Command {
	var opts pushOptions
	cmd := &cobra.Command{
		Use:   "push [<app-name>]",
		Short: "Push the application to a registry",
		Args:  cli.RequiresMaxArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPush(dockerCli, firstOrEmpty(args), opts)
		},
	}
	cmd.Flags().StringVar(&opts.namespace, "namespace", "", "Namespace to use (default: namespace in metadata)")
	cmd.Flags().StringVarP(&opts.tag, "tag", "t", "", "Tag to use (default: version in metadata)")
	cmd.Flags().StringVar(&opts.repo, "repo", "", "Name of the remote repository (default: <app-name>.dockerapp)")
	cmd.Flags().BoolVar(&opts.insecure, "insecure", false, "Use insecure registry, without SSL")
	return cmd
}

func runPush(dockerCli command.Cli, name string, opts pushOptions) error {
	app, err := packager.Extract(name)
	if err != nil {
		return err
	}
	defer app.Cleanup()

	ref := makeReference(app, opts)
	bndle, err := makeBundleFromApp(dockerCli, app, opts.namespace, "")
	if err != nil {
		return err
	}
	if err := fixupContainerImages(dockerCli, bndle); err != nil {
		return err
	}
	signedData, err := makeSignedData(bndle)
	if err != nil {
		return err
	}
	digest, err := image.PushBundle(context.TODO(), dockerCli, opts.insecure, signedData, ref)
	if err != nil {
		return err
	}
	fmt.Printf("Successfully pushed %s@%s\n", ref, digest)
	return nil
}

func fixupContainerImages(dockerCli command.Cli, bndle *bundle.Bundle) error {
	imageResolver := image.NewResolver(true, dockerCli)
	if err := bndle.FixupContainerImages(imageResolver); err != nil {
		return err
	}
	return bndle.Validate()
}

func makeSignedData(bndle *bundle.Bundle) ([]byte, error) {
	keyRingFile := duffleHome().SecretKeyRing()
	if _, err := os.Stat(keyRingFile); err != nil {
		return nil, errors.New(`duffle home has not been initialized, please run "duffle init" first`)
	}
	// Load keyring
	kr, err := signature.LoadKeyRing(keyRingFile)
	if err != nil {
		return nil, err
	}
	// Find identity
	var k *signature.Key
	all := kr.PrivateKeys()
	if len(all) == 0 {
		return nil, errors.New("no private keys found")
	}
	k = all[0]

	// Sign the file
	s := signature.NewSigner(k)
	data, err := s.Clearsign(bndle)
	if err != nil {
		return nil, err
	}
	data = append(data, '\n')
	return data, nil
}

func makeReference(app *types.App, opts pushOptions) string {
	meta := app.Metadata()

	ref := opts.repo
	if ref == "" {
		ref = meta.Name
	}
	namespace := opts.namespace
	if namespace == "" {
		namespace = meta.Namespace
	}
	version := opts.tag
	if version == "" {
		version = meta.Version
	}

	if namespace != "" {
		ref = fmt.Sprintf("%s/%s", namespace, ref)
	}
	if version != "" {
		ref = fmt.Sprintf("%s:%s", ref, version)
	}
	return ref
}
