package remotes

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/log"
	"github.com/containerd/containerd/platforms"
	"github.com/containerd/containerd/remotes"
	"github.com/deislabs/cnab-go/bundle"
	"github.com/docker/cnab-to-oci/relocation"
	"github.com/docker/distribution/reference"
	ocischemav1 "github.com/opencontainers/image-spec/specs-go/v1"
)

// FixupBundle checks that all the references are present in the referenced repository, otherwise it will mount all
// the manifests to that repository. The bundle is then patched with the new digested references.
func FixupBundle(ctx context.Context, b *bundle.Bundle, ref reference.Named, resolver remotes.Resolver, opts ...FixupOption) (relocation.ImageRelocationMap, error) {
	logger := log.G(ctx)
	logger.Debugf("Fixing up bundle %s", ref)

	// Configure the fixup and the event loop
	cfg, err := newFixupConfig(b, ref, resolver, opts...)
	if err != nil {
		return nil, err
	}

	events := make(chan FixupEvent)
	eventLoopDone := make(chan struct{})
	defer func() {
		close(events)
		// wait for all queued events to be treated
		<-eventLoopDone
	}()
	go func() {
		defer close(eventLoopDone)
		for ev := range events {
			cfg.eventCallback(ev)
		}
	}()

	// Fixup invocation images
	if len(b.InvocationImages) != 1 {
		return nil, fmt.Errorf("only one invocation image supported for bundle %q", ref)
	}

	relocationMap := relocation.ImageRelocationMap{}
	if err := fixupImage(ctx, "InvocationImage", &b.InvocationImages[0].BaseImage, relocationMap, cfg, events, cfg.invocationImagePlatformFilter); err != nil {
		return nil, err
	}
	// Fixup images
	for name, original := range b.Images {
		if err := fixupImage(ctx, name, &original.BaseImage, relocationMap, cfg, events, cfg.componentImagePlatformFilter); err != nil {
			return nil, err
		}
		b.Images[name] = original
	}

	logger.Debug("Bundle fixed")
	return relocationMap, nil
}

func fixupImage(
	ctx context.Context,
	name string,
	baseImage *bundle.BaseImage,
	relocationMap relocation.ImageRelocationMap,
	cfg fixupConfig,
	events chan<- FixupEvent,
	platformFilter platforms.Matcher) error {

	log.G(ctx).Debugf("Updating entry in relocation map for %q", baseImage.Image)
	ctx = withMutedContext(ctx)
	notifyEvent, progress := makeEventNotifier(events, baseImage.Image, cfg.targetRef)

	notifyEvent(FixupEventTypeCopyImageStart, "", nil)
	// Fixup Base image
	fixupInfo, pushed, err := fixupBaseImage(ctx, name, baseImage, cfg)
	if err != nil {
		return notifyError(notifyEvent, err)
	}
	// Update the relocation map with the original image name and the digested reference of the image pushed inside the bundle repository
	newRef, err := reference.WithDigest(fixupInfo.targetRepo, fixupInfo.resolvedDescriptor.Digest)
	if err != nil {
		return err
	}

	relocationMap[baseImage.Image] = newRef.String()

	// if the autoUpdateBundle flag is passed, mutate the bundle with the resolved digest, mediaType, and size
	if cfg.autoBundleUpdate {
		baseImage.Digest = fixupInfo.resolvedDescriptor.Digest.String()
		baseImage.Size = uint64(fixupInfo.resolvedDescriptor.Size)
		baseImage.MediaType = fixupInfo.resolvedDescriptor.MediaType
	} else {
		if baseImage.Digest != fixupInfo.resolvedDescriptor.Digest.String() {
			return fmt.Errorf("image %q digest differs %q after fixup: %q", baseImage.Image, baseImage.Digest, fixupInfo.resolvedDescriptor.Digest.String())
		}
		if baseImage.Size != uint64(fixupInfo.resolvedDescriptor.Size) {
			return fmt.Errorf("image %q size differs %d after fixup: %d", baseImage.Image, baseImage.Size, fixupInfo.resolvedDescriptor.Size)
		}
		if baseImage.MediaType != fixupInfo.resolvedDescriptor.MediaType {
			return fmt.Errorf("image %q media type differs %q after fixup: %q", baseImage.Image, baseImage.MediaType, fixupInfo.resolvedDescriptor.MediaType)
		}
	}

	if pushed {
		notifyEvent(FixupEventTypeCopyImageEnd, "Image has been pushed for service "+name, nil)
		return nil
	}

	if fixupInfo.sourceRef.Name() == fixupInfo.targetRepo.Name() {
		notifyEvent(FixupEventTypeCopyImageEnd, "Nothing to do: image reference is already present in repository"+fixupInfo.targetRepo.String(), nil)
		return nil
	}

	sourceFetcher, err := makeSourceFetcher(ctx, cfg.resolver, fixupInfo.sourceRef.Name())
	if err != nil {
		return notifyError(notifyEvent, err)
	}

	// Fixup platforms
	if err := fixupPlatforms(ctx, baseImage, relocationMap, &fixupInfo, sourceFetcher, platformFilter); err != nil {
		return notifyError(notifyEvent, err)
	}

	// Prepare and run the copier
	walkerDep, cleaner, err := makeManifestWalker(ctx, sourceFetcher, notifyEvent, cfg, fixupInfo, progress)
	if err != nil {
		return notifyError(notifyEvent, err)
	}
	defer cleaner()
	if err = walkerDep.wait(); err != nil {
		return notifyError(notifyEvent, err)
	}

	notifyEvent(FixupEventTypeCopyImageEnd, "", nil)
	return nil
}

func fixupPlatforms(ctx context.Context,
	baseImage *bundle.BaseImage,
	relocationMap relocation.ImageRelocationMap,
	fixupInfo *imageFixupInfo,
	sourceFetcher sourceFetcherAdder,
	filter platforms.Matcher) error {

	logger := log.G(ctx)
	logger.Debugf("Fixup platforms for image %v, with relocation map %v", baseImage, relocationMap)
	if filter == nil ||
		(fixupInfo.resolvedDescriptor.MediaType != ocischemav1.MediaTypeImageIndex &&
			fixupInfo.resolvedDescriptor.MediaType != images.MediaTypeDockerSchema2ManifestList) {
		// no platform filter if platform is empty, or if the descriptor is not an OCI Index / Docker Manifest list
		return nil
	}

	reader, err := sourceFetcher.Fetch(ctx, fixupInfo.resolvedDescriptor)
	if err != nil {
		return err
	}
	defer reader.Close()

	manifestBytes, err := ioutil.ReadAll(reader)
	if err != nil {
		return err
	}
	var manifestList typelessManifestList
	if err := json.Unmarshal(manifestBytes, &manifestList); err != nil {
		return err
	}
	var validManifests []typelessDescriptor
	for _, d := range manifestList.Manifests {
		if d.Platform != nil && filter.Match(*d.Platform) {
			validManifests = append(validManifests, d)
		}
	}
	if len(validManifests) == 0 {
		return fmt.Errorf("no descriptor matching the platform filter found in %q", fixupInfo.sourceRef)
	}
	manifestList.Manifests = validManifests
	manifestBytes, err = json.Marshal(&manifestList)
	if err != nil {
		return err
	}
	d := sourceFetcher.Add(manifestBytes)
	descriptor := fixupInfo.resolvedDescriptor
	descriptor.Digest = d
	descriptor.Size = int64(len(manifestBytes))
	fixupInfo.resolvedDescriptor = descriptor

	return nil
}

func fixupBaseImage(ctx context.Context, name string, baseImage *bundle.BaseImage, cfg fixupConfig) (imageFixupInfo, bool, error) {
	// Check image references
	if err := checkBaseImage(baseImage); err != nil {
		return imageFixupInfo{}, false, fmt.Errorf("invalid image %q for service %q: %s", baseImage.Image, name, err)
	}
	targetRepoOnly, err := reference.ParseNormalizedNamed(cfg.targetRef.Name())
	if err != nil {
		return imageFixupInfo{}, false, err
	}

	var (
		descriptor     ocischemav1.Descriptor
		pushed         bool
		sourceImageRef reference.Named
	)

	if baseImage.Image == "" && cfg.pushImages {
		// no image is defined for the service, try to push by digest from local docker image store
		descriptor, err = pushImageToTarget(ctx, baseImage.Digest, cfg)
		if err != nil {
			return imageFixupInfo{}, false, err
		}
		pushed = true
	} else {
		// try to resolve
		sourceImageRef, err = reference.ParseNormalizedNamed(baseImage.Image)
		if err != nil {
			return imageFixupInfo{}, false, fmt.Errorf("%q is not a valid image reference for %q: %s", baseImage.Image, cfg.targetRef, err)
		}
		sourceImageRef = reference.TagNameOnly(sourceImageRef)

		// Try to fetch the image descriptor
		_, descriptor, err = cfg.resolver.Resolve(ctx, sourceImageRef.String())
		if err != nil {
			if cfg.pushImages {
				// try to push from local docker image store
				descriptor, err = pushImageToTarget(ctx, baseImage.Image, cfg)
				if err != nil {
					return imageFixupInfo{}, false, err
				}
				pushed = true
			} else {
				return imageFixupInfo{}, false, fmt.Errorf("failed to resolve %q, push the image for service %q to the registry before pushing the bundle: %s", sourceImageRef, name, err)
			}
		}
	}
	return imageFixupInfo{
		targetRepo:         targetRepoOnly,
		sourceRef:          sourceImageRef,
		resolvedDescriptor: descriptor,
	}, pushed, nil
}

// pushImageToTarget pushes the image from the local docker daemon store to the target defined in the configuration.
// Docker image cannot be pushed by digest to a registry. So to be able to push the image inside the targeted repository
// the same behaviour than for multi architecture images is used: all the images are tagged for the targeted repository
// and then pushed.
// Every time a new image is pushed under a tag, the previous tagged image will be untagged. But this untagged image
// remains accessible using its digest. So right after pushing it, the image is resolved to grab its digest from the
// registry and can be added to the index.
// The final workflow is then:
//  - tag the image to push with targeted reference
//  - push the image using a docker `ImageAPIClient`
//  - resolve the pushed image to grab its digest
func pushImageToTarget(ctx context.Context, src string, cfg fixupConfig) (ocischemav1.Descriptor, error) {
	taggedRef := reference.TagNameOnly(cfg.targetRef)

	if err := cfg.imageClient.ImageTag(ctx, src, cfg.targetRef.String()); err != nil {
		return ocischemav1.Descriptor{}, fmt.Errorf("failed to push image %q, make sure the image exists locally: %s", src, err)
	}

	if err := pushTaggedImage(ctx, cfg.imageClient, cfg.targetRef, cfg.pushOut); err != nil {
		return ocischemav1.Descriptor{}, fmt.Errorf("failed to push image %q: %s", src, err)
	}

	_, descriptor, err := cfg.resolver.Resolve(ctx, taggedRef.String())
	if err != nil {
		return ocischemav1.Descriptor{}, fmt.Errorf("failed to resolve %q after pushing it: %s", taggedRef, err)
	}

	return descriptor, nil
}
