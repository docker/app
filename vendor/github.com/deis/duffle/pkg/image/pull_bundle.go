package image

import (
	"context"
	"io/ioutil"

	"github.com/docker/cli/cli/command"
)

// PullBundle pulls a signed bundle from an image registry.
func PullBundle(ctx context.Context, cli command.Cli, nonSSL bool, ref string) ([]byte, error) {
	named, repoName, tag, err := parseBundleReference(ref)
	if err != nil {
		return nil, err
	}

	regClient, err := makeRegClient(ctx, cli, nonSSL, named)
	if err != nil {
		return nil, err
	}

	manifest, err := regClient.ManifestV2(repoName, tag)
	if err != nil {
		return nil, err
	}

	// TODO: Check Mediatype once hub accepts bundle media type
	digest := manifest.Config.Digest

	reader, err := regClient.DownloadLayer(repoName, digest)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return ioutil.ReadAll(reader)
}
