package main

import (
	"context"
	"fmt"
	"io/ioutil"

	"github.com/deislabs/duffle/pkg/bundle"
	"github.com/docker/app/internal/packager"
	"github.com/docker/app/types/metadata"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cnab-to-oci/remotes"
	"github.com/docker/distribution/reference"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/registry"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type pushOptions struct {
	registry registryOptions
	tag      string
}

func pushCmd(dockerCli command.Cli) *cobra.Command {
	var opts pushOptions
	cmd := &cobra.Command{
		Use:   "push [<app-name>]",
		Short: "Push the application to a registry",
		Args:  cli.RequiresMaxArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := ""
			if len(args) > 0 {
				name = args[0]
			}
			return runPush(dockerCli, name, opts)
		},
	}
	flags := cmd.Flags()
	flags.StringVarP(&opts.tag, "tag", "t", "", "Target registry reference (default is : <name>:<version> from metadata)")
	opts.registry.addFlags(flags)
	return cmd
}

func runPush(dockerCli command.Cli, name string, opts pushOptions) error {
	muteDockerCli(dockerCli)
	app, err := packager.Extract(name)
	if err != nil {
		return err
	}
	defer app.Cleanup()
	bndl, err := makeBundleFromApp(dockerCli, app)
	if err != nil {
		return err
	}
	retag, err := shouldRetagInvocationImage(app.Metadata(), bndl, opts.tag)
	if err != nil {
		return err
	}
	if retag.shouldRetag {
		err := retagInvocationImage(dockerCli, bndl, retag.invocationImageRef.String())
		if err != nil {
			return err
		}
	}

	// pushing invocation image
	repoInfo, err := registry.ParseRepositoryInfo(retag.invocationImageRef)
	if err != nil {
		return err
	}
	encodedAuth, err := command.EncodeAuthToBase64(command.ResolveAuthConfig(context.Background(), dockerCli, repoInfo.Index))
	if err != nil {
		return err
	}
	reader, err := dockerCli.Client().ImagePush(context.Background(), retag.invocationImageRef.String(), types.ImagePushOptions{
		RegistryAuth: encodedAuth,
	})
	if err != nil {
		return err
	}
	defer reader.Close()
	if err = jsonmessage.DisplayJSONMessagesStream(reader, ioutil.Discard, 0, false, nil); err != nil {
		return err
	}

	dockerResolver := remotes.CreateResolver(dockerCli.ConfigFile(), opts.registry.insecureRegistries...)
	// bundle fixup
	if err := remotes.FixupBundle(context.Background(), bndl, retag.cnabRef, dockerResolver); err != nil {
		return err
	}
	// push bundle manifest
	descriptor, err := remotes.Push(context.Background(), bndl, retag.cnabRef, dockerResolver)
	if err != nil {
		return err
	}
	fmt.Printf("Successfully pushed bundle to %s. Digest is %s.\n", retag.cnabRef.String(), descriptor.Digest)
	return nil
}

func retagInvocationImage(dockerCli command.Cli, bndl *bundle.Bundle, newName string) error {
	err := dockerCli.Client().ImageTag(context.Background(), bndl.InvocationImages[0].Image, newName)
	if err != nil {
		return err
	}
	bndl.InvocationImages[0].Image = newName
	return nil
}

type retagResult struct {
	shouldRetag        bool
	cnabRef            reference.Named
	invocationImageRef reference.Named
}

func shouldRetagInvocationImage(meta metadata.AppMetadata, bndl *bundle.Bundle, tagOverride string) (retagResult, error) {
	imgName := tagOverride
	var err error
	if imgName == "" {
		imgName, err = makeCNABImageName(meta, "")
		if err != nil {
			return retagResult{}, err
		}
	}
	cnabRef, err := reference.ParseNormalizedNamed(imgName)
	if err != nil {
		return retagResult{}, errors.Wrap(err, imgName)
	}
	if _, digested := cnabRef.(reference.Digested); digested {
		return retagResult{}, errors.Errorf("%s: can't push to a digested reference", cnabRef)
	}
	cnabRef = reference.TagNameOnly(cnabRef)
	expectedInvocationImageRef, err := reference.ParseNormalizedNamed(reference.TagNameOnly(cnabRef).String() + "-invoc")
	if err != nil {
		return retagResult{}, errors.Wrap(err, reference.TagNameOnly(cnabRef).String()+"-invoc")
	}
	currentInvocationImageRef, err := reference.ParseNormalizedNamed(bndl.InvocationImages[0].Image)
	if err != nil {
		return retagResult{}, errors.Wrap(err, bndl.InvocationImages[0].Image)
	}
	return retagResult{
		cnabRef:            cnabRef,
		invocationImageRef: expectedInvocationImageRef,
		shouldRetag:        expectedInvocationImageRef.String() != currentInvocationImageRef.String(),
	}, nil
}
