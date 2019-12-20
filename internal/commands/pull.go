package commands

import (
	"context"
	"fmt"
	"os"

	"github.com/docker/app/internal/cnab"
	"github.com/docker/app/internal/packager"
	"github.com/docker/app/internal/store"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/config"
	"github.com/docker/distribution/reference"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/registry"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type pullOptions struct {
	serviceImages bool
}

func pullCmd(dockerCli command.Cli) *cobra.Command {
	var opts pullOptions

	cmd := &cobra.Command{
		Use:     "pull [OPTIONS] APP_IMAGE",
		Short:   "Pull an App image from a registry",
		Example: `$ docker app pull myrepo/myapp:0.1.0`,
		Args:    cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPull(dockerCli, opts, args[0])
		},
	}
	cmd.Flags().BoolVar(&opts.serviceImages, "service-images", false, "Also pull down service images to this host's context")
	return cmd
}

func pullImage(ctx context.Context, cli command.Cli, image string) error {
	ref, err := reference.ParseNormalizedNamed(image)
	if err != nil {
		return err
	}

	// Resolve the Repository name from fqn to RepositoryInfo
	repoInfo, err := registry.ParseRepositoryInfo(ref)
	if err != nil {
		return err
	}
	authConfig := command.ResolveAuthConfig(ctx, cli, repoInfo.Index)
	encodedAuth, err := command.EncodeAuthToBase64(authConfig)
	if err != nil {
		return err
	}
	options := types.ImagePullOptions{
		RegistryAuth: encodedAuth,
	}
	responseBody, err := cli.Client().ImagePull(ctx, image, options)
	if err != nil {
		return err
	}
	defer responseBody.Close()

	return jsonmessage.DisplayJSONMessagesStream(responseBody, cli.Out(), cli.Out().FD(), false, nil)
}

func runPull(dockerCli command.Cli, opts pullOptions, name string) error {
	appstore, err := store.NewApplicationStore(config.Dir())
	if err != nil {
		return err
	}
	imageStore, err := appstore.ImageStore()
	if err != nil {
		return err
	}
	ref, err := reference.ParseNormalizedNamed(name)
	if err != nil {
		return errors.Wrap(err, name)
	}
	tagRef := reference.TagNameOnly(ref)

	bndl, err := cnab.PullBundle(dockerCli, imageStore, tagRef)
	if err != nil {
		return errors.Wrap(err, name)
	}
	if err := packager.CheckAppVersion(dockerCli.Err(), bndl.Bundle); err != nil {
		return err
	}
	fmt.Fprintf(os.Stdout, "Successfully pulled %q (%s) from %s\n", bndl.Name, bndl.Version, ref.String())

	if opts.serviceImages {
		ctx := context.Background()
		for name, image := range bndl.Images {
			fmt.Fprintf(os.Stdout, "Pulling: %s -> %s\n", name, image.Image)
			if err := pullImage(ctx, dockerCli, image.Image); err != nil {
				return err
			}
		}
	}

	return nil
}
