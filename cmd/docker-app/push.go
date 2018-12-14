package main

import (
	"context"
	"fmt"

	"github.com/deislabs/duffle/pkg/bundle"
	"github.com/deislabs/duffle/pkg/image"
	"github.com/docker/app/internal/packager"
	"github.com/docker/app/types"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cnab-to-oci/remotes"
	"github.com/docker/distribution/reference"
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
	named, err := reference.ParseNormalizedNamed(ref)
	if err != nil {
		return err
	}
	bndle, err := makeBundleFromApp(dockerCli, app, opts.namespace, "")
	if err != nil {
		return err
	}
	if err := fixupContainerImages(dockerCli, bndle); err != nil {
		return err
	}
	resolver := remotes.CreateResolver(dockerCli.ConfigFile(), opts.insecure)
	if err := remotes.FixupBundle(context.Background(), bndle, named, resolver); err != nil {
		return err
	}
	descriptor, err := remotes.Push(context.Background(), bndle, named, resolver)
	if err != nil {
		return err
	}
	fmt.Printf("Successfully pushed %s@%s\n", ref, descriptor.Digest)
	return nil
}

func fixupContainerImages(dockerCli command.Cli, bndle *bundle.Bundle) error {
	imageResolver := image.NewResolver(true, dockerCli)
	for ix, invocImage := range bndle.InvocationImages {
		if invocImage.ImageType != "" &&
			invocImage.ImageType != "docker" &&
			invocImage.Image != "oci" {
			continue
		}
		named, err := reference.ParseNormalizedNamed(invocImage.Image)
		if err != nil {
			return err
		}
		var dig string
		if digested, ok := named.(reference.Digested); ok {
			dig = digested.Digest().String()
		}
		fixedUpRef, _, err := imageResolver.Resolve(invocImage.Image, dig)
		if err != nil {
			return err
		}
		bndle.InvocationImages[ix].Image = fixedUpRef
	}
	return nil
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
