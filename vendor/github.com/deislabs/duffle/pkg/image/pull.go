package image

import (
	"context"

	"github.com/docker/cli/cli/command"
	"github.com/docker/distribution/reference"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/registry"
)

// PullImage pulls an docker image
func PullImage(ctx context.Context, cli command.Cli, image string) error {
	repoInfo, err := getRepoInfo(image)
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
	return jsonmessage.DisplayJSONMessagesStream(
		responseBody,
		cli.Out(),
		cli.Out().FD(),
		cli.Out().IsTerminal(),
		nil)
}

func getRepoInfo(imageName string) (*registry.RepositoryInfo, error) {
	ref, err := reference.ParseNormalizedNamed(imageName)
	if err != nil {
		return nil, err
	}

	// Resolve the Repository name from fqn to RepositoryInfo
	return registry.ParseRepositoryInfo(ref)
}

func pushImage(ctx context.Context, cli command.Cli, image string) error {
	repoInfo, err := getRepoInfo(image)
	if err != nil {
		return err
	}
	authConfig := command.ResolveAuthConfig(ctx, cli, repoInfo.Index)
	encodedAuth, err := command.EncodeAuthToBase64(authConfig)
	if err != nil {
		return err
	}
	requestPrivilege := command.RegistryAuthenticationPrivilegedFunc(cli, repoInfo.Index, "push")
	options := types.ImagePushOptions{
		RegistryAuth:  encodedAuth,
		PrivilegeFunc: requestPrivilege,
	}
	responseBody, err := cli.Client().ImagePush(ctx, image, options)
	if err != nil {
		return err
	}
	defer responseBody.Close()
	return jsonmessage.DisplayJSONMessagesStream(
		responseBody,
		cli.Out(),
		cli.Out().FD(),
		cli.Out().IsTerminal(),
		nil)
}
