package remotes

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/docker/cnab-to-oci/internal"

	"github.com/docker/cli/cli/config"
	configtypes "github.com/docker/cli/cli/config/types"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/registry"

	"github.com/containerd/containerd/errdefs"
	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/log"
	"github.com/containerd/containerd/remotes"
	"github.com/deislabs/cnab-go/bundle"
	"github.com/docker/cnab-to-oci/converter"
	"github.com/docker/cnab-to-oci/relocation"
	"github.com/docker/distribution/reference"
	registrytypes "github.com/docker/docker/api/types/registry"
	"github.com/opencontainers/go-digest"
	ocischemav1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
)

// ManifestOption is a callback used to customize a manifest before pushing it
type ManifestOption func(*ocischemav1.Index) error

// Push pushes a bundle as an OCI Image Index manifest
func Push(ctx context.Context,
	b *bundle.Bundle,
	relocationMap relocation.ImageRelocationMap,
	ref reference.Named,
	resolver remotes.Resolver,
	allowFallbacks bool,
	options ...ManifestOption) (ocischemav1.Descriptor, error) {
	log.G(ctx).Debugf("Pushing CNAB Bundle %s", ref)

	confManifestDescriptor, err := pushConfig(ctx, b, ref, resolver, allowFallbacks)
	if err != nil {
		return ocischemav1.Descriptor{}, err
	}

	indexDescriptor, err := pushIndex(ctx, b, relocationMap, ref, resolver, allowFallbacks, confManifestDescriptor, options...)
	if err != nil {
		return ocischemav1.Descriptor{}, err
	}

	log.G(ctx).Debug("CNAB Bundle pushed")
	return indexDescriptor, nil
}

func pushConfig(ctx context.Context,
	b *bundle.Bundle,
	ref reference.Named, //nolint:interfacer
	resolver remotes.Resolver,
	allowFallbacks bool) (ocischemav1.Descriptor, error) {
	logger := log.G(ctx)
	logger.Debugf("Pushing CNAB Bundle Config")

	bundleConfig, err := converter.PrepareForPush(b)
	if err != nil {
		return ocischemav1.Descriptor{}, err
	}
	confManifestDescriptor, err := pushBundleConfig(ctx, resolver, ref.Name(), bundleConfig, allowFallbacks)
	if err != nil {
		return ocischemav1.Descriptor{}, fmt.Errorf("error while pushing bundle config manifest: %s", err)
	}

	logger.Debug("CNAB Bundle Config pushed")
	return confManifestDescriptor, nil
}

func pushIndex(ctx context.Context, b *bundle.Bundle, relocationMap relocation.ImageRelocationMap, ref reference.Named, resolver remotes.Resolver, allowFallbacks bool,
	confManifestDescriptor ocischemav1.Descriptor, options ...ManifestOption) (ocischemav1.Descriptor, error) {
	logger := log.G(ctx)
	logger.Debug("Pushing CNAB Index")

	indexDescriptor, indexPayload, err := prepareIndex(b, relocationMap, ref, confManifestDescriptor, options...)
	if err != nil {
		return ocischemav1.Descriptor{}, err
	}
	// Push the bundle index
	logger.Debug("Trying to push OCI Index")
	logger.Debug(string(indexPayload))
	logger.Debug("OCI Index Descriptor")
	logPayload(logger, indexDescriptor)

	if err := pushPayload(ctx, resolver, ref.String(), indexDescriptor, indexPayload); err != nil {
		if !allowFallbacks {
			logger.Debug("Not using fallbacks, giving up")
			return ocischemav1.Descriptor{}, err
		}
		logger.Debugf("Unable to push OCI Index: %v", err)
		// retry with a docker manifestlist
		return pushDockerManifestList(ctx, b, relocationMap, ref, resolver, confManifestDescriptor, options...)
	}

	logger.Debugf("CNAB Index pushed")
	return indexDescriptor, nil
}

func pushDockerManifestList(ctx context.Context, b *bundle.Bundle, relocationMap relocation.ImageRelocationMap, ref reference.Named, resolver remotes.Resolver,
	confManifestDescriptor ocischemav1.Descriptor, options ...ManifestOption) (ocischemav1.Descriptor, error) {
	logger := log.G(ctx)

	indexDescriptor, indexPayload, err := prepareIndexNonOCI(b, relocationMap, ref, confManifestDescriptor, options...)
	if err != nil {
		return ocischemav1.Descriptor{}, err
	}
	logger.Debug("Trying to push Index with Manifest list as fallback")
	logger.Debug(string(indexPayload))
	logger.Debug("Manifest list Descriptor")
	logPayload(logger, indexDescriptor)

	if err := pushPayload(ctx,
		resolver, ref.String(),
		indexDescriptor,
		indexPayload); err != nil {
		return ocischemav1.Descriptor{}, err
	}
	return indexDescriptor, nil
}

func prepareIndex(b *bundle.Bundle,
	relocationMap relocation.ImageRelocationMap,
	ref reference.Named,
	confDescriptor ocischemav1.Descriptor,
	options ...ManifestOption) (ocischemav1.Descriptor, []byte, error) {
	ix, err := convertIndexAndApplyOptions(b, relocationMap, ref, confDescriptor, options...)
	if err != nil {
		return ocischemav1.Descriptor{}, nil, err
	}
	indexPayload, err := json.Marshal(ix)
	if err != nil {
		return ocischemav1.Descriptor{}, nil, fmt.Errorf("invalid bundle manifest %q: %s", ref, err)
	}
	indexDescriptor := ocischemav1.Descriptor{
		Digest:    digest.FromBytes(indexPayload),
		MediaType: ocischemav1.MediaTypeImageIndex,
		Size:      int64(len(indexPayload)),
	}
	return indexDescriptor, indexPayload, nil
}

type ociIndexWrapper struct {
	ocischemav1.Index
	MediaType string `json:"mediaType,omitempty"`
}

func convertIndexAndApplyOptions(b *bundle.Bundle,
	relocationMap relocation.ImageRelocationMap,
	ref reference.Named,
	confDescriptor ocischemav1.Descriptor,
	options ...ManifestOption) (*ocischemav1.Index, error) {
	ix, err := converter.ConvertBundleToOCIIndex(b, ref, confDescriptor, relocationMap)
	if err != nil {
		return nil, err
	}
	for _, opts := range options {
		if err := opts(ix); err != nil {
			return nil, fmt.Errorf("failed to prepare bundle manifest %q: %s", ref, err)
		}
	}
	return ix, nil
}

func prepareIndexNonOCI(b *bundle.Bundle,
	relocationMap relocation.ImageRelocationMap,
	ref reference.Named,
	confDescriptor ocischemav1.Descriptor,
	options ...ManifestOption) (ocischemav1.Descriptor, []byte, error) {
	ix, err := convertIndexAndApplyOptions(b, relocationMap, ref, confDescriptor, options...)
	if err != nil {
		return ocischemav1.Descriptor{}, nil, err
	}
	w := &ociIndexWrapper{Index: *ix, MediaType: images.MediaTypeDockerSchema2ManifestList}
	w.SchemaVersion = 2
	indexPayload, err := json.Marshal(w)
	if err != nil {
		return ocischemav1.Descriptor{}, nil, fmt.Errorf("invalid bundle manifest %q: %s", ref, err)
	}
	indexDescriptor := ocischemav1.Descriptor{
		Digest:    digest.FromBytes(indexPayload),
		MediaType: images.MediaTypeDockerSchema2ManifestList,
		Size:      int64(len(indexPayload)),
	}
	return indexDescriptor, indexPayload, nil
}

func pushPayload(ctx context.Context, resolver remotes.Resolver, reference string, descriptor ocischemav1.Descriptor, payload []byte) error {
	ctx = withMutedContext(ctx)
	pusher, err := resolver.Pusher(ctx, reference)
	if err != nil {
		return err
	}
	writer, err := pusher.Push(ctx, descriptor)
	if err != nil {
		if errors.Cause(err) == errdefs.ErrAlreadyExists {
			return nil
		}
		return err
	}
	defer writer.Close()
	if _, err := writer.Write(payload); err != nil {
		if errors.Cause(err) == errdefs.ErrAlreadyExists {
			return nil
		}
		return err
	}
	err = writer.Commit(ctx, descriptor.Size, descriptor.Digest)
	if errors.Cause(err) == errdefs.ErrAlreadyExists {
		return nil
	}
	return err
}

func pushBundleConfig(ctx context.Context, resolver remotes.Resolver, reference string, bundleConfig *converter.PreparedBundleConfig, allowFallbacks bool) (ocischemav1.Descriptor, error) {
	if d, err := pushBundleConfigDescriptor(ctx, "Config", resolver, reference,
		bundleConfig.ConfigBlobDescriptor, bundleConfig.ConfigBlob, bundleConfig.Fallback, allowFallbacks); err != nil {
		return d, err
	}
	return pushBundleConfigDescriptor(ctx, "Config Manifest", resolver, reference,
		bundleConfig.ManifestDescriptor, bundleConfig.Manifest, bundleConfig.Fallback, allowFallbacks)
}

func pushBundleConfigDescriptor(ctx context.Context, name string, resolver remotes.Resolver, reference string,
	descriptor ocischemav1.Descriptor, payload []byte, fallback *converter.PreparedBundleConfig, allowFallbacks bool) (ocischemav1.Descriptor, error) {
	logger := log.G(ctx)
	logger.Debugf("Trying to push CNAB Bundle %s", name)
	logger.Debugf("CNAB Bundle %s Descriptor", name)
	logPayload(logger, descriptor)

	if err := pushPayload(ctx, resolver, reference, descriptor, payload); err != nil {
		if allowFallbacks && fallback != nil {
			logger.Debugf("Failed to push CNAB Bundle %s, trying with a fallback method", name)
			return pushBundleConfig(ctx, resolver, reference, fallback, allowFallbacks)
		}
		return ocischemav1.Descriptor{}, err
	}
	return descriptor, nil
}

func pushTaggedImage(ctx context.Context, imageClient internal.ImageClient, targetRef reference.Named, out io.Writer) error {
	repoInfo, err := registry.ParseRepositoryInfo(targetRef)
	if err != nil {
		return err
	}

	authConfig := resolveAuthConfig(repoInfo.Index)
	encodedAuth, err := encodeAuthToBase64(authConfig)
	if err != nil {
		return err
	}

	reader, err := imageClient.ImagePush(ctx, targetRef.String(), types.ImagePushOptions{
		RegistryAuth: encodedAuth,
	})
	if err != nil {
		return err
	}
	defer reader.Close()
	return jsonmessage.DisplayJSONMessagesStream(reader, out, 0, false, nil)
}

func encodeAuthToBase64(authConfig configtypes.AuthConfig) (string, error) {
	buf, err := json.Marshal(authConfig)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(buf), nil
}

func resolveAuthConfig(index *registrytypes.IndexInfo) configtypes.AuthConfig {
	cfg := config.LoadDefaultConfigFile(os.Stderr)

	hostName := index.Name
	if index.Official {
		hostName = registry.IndexServer
	}

	configs, err := cfg.GetAllCredentials()
	if err != nil {
		return configtypes.AuthConfig{}
	}

	authConfig, ok := configs[hostName]
	if !ok {
		return configtypes.AuthConfig{}
	}
	return authConfig
}
