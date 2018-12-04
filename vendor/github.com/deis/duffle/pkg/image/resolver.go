package image

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/docker/cli/cli/command"
	"github.com/docker/docker/client"
)

// Resolver implements bundle.ContainerImageResolver
type Resolver struct {
	dockerCli       command.Cli
	pushLocalImages bool
}

func resolve(cli command.Cli, image, digest string, pushLocalImages bool) (string, string, error) {
	if digest != "" {
		// The digest is already there, just append it to the image name
		if !strings.HasSuffix(image, "@"+digest) {
			image = fmt.Sprintf("%s@%s", image, digest)
		}
		return image, digest, nil
	}
	ctx := context.Background()
	// Inspect local images to retrieve the digest, pull from registry if not found
	result, _, err := cli.Client().ImageInspectWithRaw(ctx, image)
	if client.IsErrNotFound(err) {
		// Try to pull image
		if err := PullImage(ctx, cli, image); err != nil {
			return "", "", err
		}
		if result, _, err = cli.Client().ImageInspectWithRaw(ctx, image); err != nil {
			return "", "", err
		}
	} else if err != nil {
		return "", "", err
	}

	digestedRef, err := getFirstMatchingDigest(image, result.RepoDigests)
	if err != nil {
		if pushLocalImages {
			if err := pushImage(ctx, cli, image); err != nil {
				return "", "", err
			}
			return resolve(cli, image, digest, false)
		}
		return "", "", imageLocalOnlyError{name: image}
	}
	return digestedRef, strings.Split(digestedRef, "@")[1], nil
}

// Resolve resolves an image and return the digested reference and digest
func (r *Resolver) Resolve(image, digest string) (string, string, error) {
	return resolve(r.dockerCli, image, digest, r.pushLocalImages)
}

// NewResolver creates a container image resolver
func NewResolver(pushLocalImages bool, dockerCli command.Cli) *Resolver {
	return &Resolver{dockerCli: dockerCli, pushLocalImages: pushLocalImages}
}

func getFirstMatchingDigest(image string, digestedRefs []string) (string, error) {
	repoInfo, err := getRepoInfo(image)
	if err != nil {
		return "", err
	}
	for _, candidate := range digestedRefs {
		candidateRepoInfo, err := getRepoInfo(candidate)
		if err != nil {
			return "", err
		}
		if candidateRepoInfo.Index.Name == repoInfo.Index.Name {
			return candidate, nil
		}
	}
	return "", errors.New("not found")
}

type imageLocalOnlyError struct {
	name string
}

func (e imageLocalOnlyError) Error() string {
	return fmt.Sprintf("image %q has no repository digest", e.name)
}

// IsErrImageLocalOnly indicates if the error is about an image with no repository digest (image built locally)
func IsErrImageLocalOnly(err error) (bool, string) {
	e, ok := err.(imageLocalOnlyError)
	if ok {
		return ok, e.name
	}
	return false, ""
}
