package remotes

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/log"
	"github.com/containerd/containerd/remotes"
	"github.com/deislabs/cnab-go/bundle"
	"github.com/docker/cli/opts"
	"github.com/docker/cnab-to-oci/converter"
	"github.com/docker/cnab-to-oci/relocation"
	"github.com/docker/distribution/reference"
	"github.com/docker/distribution/registry/client/auth"
	ocischemav1 "github.com/opencontainers/image-spec/specs-go/v1"
)

// Pull pulls a bundle from an OCI Image Index manifest
func Pull(ctx context.Context, ref reference.Named, resolver remotes.Resolver) (*bundle.Bundle, relocation.ImageRelocationMap, error) {
	log.G(ctx).Debugf("Pulling CNAB Bundle %s", ref)
	index, err := getIndex(ctx, ref, resolver)
	if err != nil {
		return nil, nil, err
	}
	b, err := getBundle(ctx, ref, resolver, index)
	if err != nil {
		return nil, nil, err
	}
	relocationMap, err := converter.GenerateRelocationMap(&index, b, ref)
	if err != nil {
		return nil, nil, err
	}
	return b, relocationMap, nil
}

func getIndex(ctx context.Context, ref auth.Scope, resolver remotes.Resolver) (ocischemav1.Index, error) {
	logger := log.G(ctx)

	logger.Debug("Getting OCI Index Descriptor")
	resolvedRef, indexDescriptor, err := resolver.Resolve(withMutedContext(ctx), ref.String())
	if err != nil {
		return ocischemav1.Index{}, fmt.Errorf("failed to resolve bundle manifest %q: %s", ref, err)
	}
	if indexDescriptor.MediaType != ocischemav1.MediaTypeImageIndex && indexDescriptor.MediaType != images.MediaTypeDockerSchema2ManifestList {
		return ocischemav1.Index{}, fmt.Errorf("invalid media type %q for bundle manifest", indexDescriptor.MediaType)
	}
	logPayload(logger, indexDescriptor)

	logger.Debugf("Fetching OCI Index %s", indexDescriptor.Digest)
	indexPayload, err := pullPayload(ctx, resolver, resolvedRef, indexDescriptor)
	if err != nil {
		return ocischemav1.Index{}, fmt.Errorf("failed to pull bundle manifest %q: %s", ref, err)
	}
	var index ocischemav1.Index
	if err := json.Unmarshal(indexPayload, &index); err != nil {
		return ocischemav1.Index{}, fmt.Errorf("failed to pull bundle manifest %q: %s", ref, err)
	}
	logPayload(logger, index)

	return index, nil
}

func getBundle(ctx context.Context, ref opts.NamedOption, resolver remotes.Resolver, index ocischemav1.Index) (*bundle.Bundle, error) {
	repoOnly, err := reference.ParseNormalizedNamed(ref.Name())
	if err != nil {
		return nil, fmt.Errorf("invalid bundle manifest reference name %q: %s", ref, err)
	}

	// config is wrapped in an image manifest. So we first pull the manifest
	// and then the config blob within it
	configManifestDescriptor, err := getConfigManifestDescriptor(ctx, ref, index)
	if err != nil {
		return nil, err
	}

	manifest, err := getConfigManifest(ctx, ref, repoOnly, resolver, configManifestDescriptor)
	if err != nil {
		return nil, err
	}

	// Pull now the bundle itself
	return getBundleConfig(ctx, ref, repoOnly, resolver, manifest)
}

func getConfigManifestDescriptor(ctx context.Context, ref opts.NamedOption, index ocischemav1.Index) (ocischemav1.Descriptor, error) {
	logger := log.G(ctx)

	logger.Debug("Getting Bundle Config Manifest Descriptor")
	configManifestDescriptor, err := converter.GetBundleConfigManifestDescriptor(&index)
	if err != nil {
		return ocischemav1.Descriptor{}, fmt.Errorf("failed to get bundle config manifest from %q: %s", ref, err)
	}
	logPayload(logger, configManifestDescriptor)

	return configManifestDescriptor, nil
}

func getConfigManifest(ctx context.Context, ref opts.NamedOption, repoOnly reference.Named, resolver remotes.Resolver, configManifestDescriptor ocischemav1.Descriptor) (ocischemav1.Manifest, error) {
	logger := log.G(ctx)

	logger.Debugf("Getting Bundle Config Manifest %s", configManifestDescriptor.Digest)
	configManifestRef, err := reference.WithDigest(repoOnly, configManifestDescriptor.Digest)
	if err != nil {
		return ocischemav1.Manifest{}, fmt.Errorf("invalid bundle config manifest reference name %q: %s", ref, err)
	}
	configManifestPayload, err := pullPayload(ctx, resolver, configManifestRef.String(), configManifestDescriptor)
	if err != nil {
		return ocischemav1.Manifest{}, fmt.Errorf("failed to pull bundle config manifest %q: %s", ref, err)
	}
	var manifest ocischemav1.Manifest
	if err := json.Unmarshal(configManifestPayload, &manifest); err != nil {
		return ocischemav1.Manifest{}, err
	}
	logPayload(logger, manifest)

	return manifest, err
}

func getBundleConfig(ctx context.Context, ref opts.NamedOption, repoOnly reference.Named, resolver remotes.Resolver, manifest ocischemav1.Manifest) (*bundle.Bundle, error) {
	logger := log.G(ctx)

	logger.Debugf("Fetching Bundle %s", manifest.Config.Digest)
	configRef, err := reference.WithDigest(repoOnly, manifest.Config.Digest)
	if err != nil {
		return nil, fmt.Errorf("invalid bundle reference name %q: %s", ref, err)
	}
	configPayload, err := pullPayload(ctx, resolver, configRef.String(), ocischemav1.Descriptor{
		Digest:    manifest.Config.Digest,
		MediaType: manifest.Config.MediaType,
		Size:      manifest.Config.Size,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to pull bundle %q: %s", ref, err)
	}
	var b bundle.Bundle
	if err := json.Unmarshal(configPayload, &b); err != nil {
		return nil, fmt.Errorf("failed to pull bundle %q: %s", ref, err)
	}
	logPayload(logger, b)

	return &b, nil
}

func pullPayload(ctx context.Context, resolver remotes.Resolver, reference string, descriptor ocischemav1.Descriptor) ([]byte, error) {
	ctx = withMutedContext(ctx)
	fetcher, err := resolver.Fetcher(ctx, reference)
	if err != nil {
		return nil, err
	}
	reader, err := fetcher.Fetch(ctx, descriptor)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	result, err := ioutil.ReadAll(reader)
	return result, err
}
