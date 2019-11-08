package internal

import (
	"context"
	"io"

	"github.com/docker/docker/api/types"
)

// ImageClient is a subset of Docker's ImageAPIClient interface with only what we are using for cnab-to-oci.
type ImageClient interface {
	ImagePush(ctx context.Context, ref string, options types.ImagePushOptions) (io.ReadCloser, error)
	ImageTag(ctx context.Context, image, ref string) error
}
