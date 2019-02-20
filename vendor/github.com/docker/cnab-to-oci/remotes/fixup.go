package remotes

import (
	"context"
	"fmt"
	"os"

	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/remotes/docker"
	"github.com/deislabs/duffle/pkg/bundle"
	"github.com/docker/cli/opts"
	"github.com/docker/distribution/reference"
	ocischemav1 "github.com/opencontainers/image-spec/specs-go/v1"
)

// FixupBundle checks that all the references are present in the referenced repository, otherwise it will mount all
// the manifests to that repository. The bundle is then patched with the new digested references.
func FixupBundle(ctx context.Context, b *bundle.Bundle, ref reference.Named, resolver docker.ResolverBlobMounter) error {
	if len(b.InvocationImages) != 1 {
		return fmt.Errorf("only one invocation image supported for bundle %q", ref)
	}
	var err error
	if b.InvocationImages[0].BaseImage, err = fixupImage(ctx, b.InvocationImages[0].BaseImage, ref, resolver); err != nil {
		return err
	}
	for name, original := range b.Images {
		if original.BaseImage, err = fixupImage(ctx, original.BaseImage, ref, resolver); err != nil {
			return err
		}
		b.Images[name] = original
	}
	return nil
}

func fixupImage(ctx context.Context, baseImage bundle.BaseImage, ref reference.Named, resolver docker.ResolverBlobMounter) (bundle.BaseImage, error) {
	fmt.Fprintf(os.Stderr, "Ensuring image %s is present in repository %s\n", baseImage.Image, ref.Name())
	repoOnly, imageRef, descriptor, err := fixupBaseImage(ctx, &baseImage, ref, resolver)
	if err != nil {
		return bundle.BaseImage{}, err
	}
	if imageRef.Name() == ref.Name() {
		return baseImage, nil
	}

	fmt.Fprintln(os.Stderr, "Image is not present in repository")
	sourceRepoOnly, err := reference.ParseNormalizedNamed(imageRef.Name())
	if err != nil {
		return bundle.BaseImage{}, err
	}
	sourceFetcher, err := resolver.Fetcher(ctx, sourceRepoOnly.Name())
	if err != nil {
		return bundle.BaseImage{}, err
	}

	// Prepare the copier or the mounter
	copier, err := newImageCopier(ctx, resolver, sourceFetcher, repoOnly.String())
	if err != nil {
		return bundle.BaseImage{}, err
	}
	var handler imageHandler = &copier
	if reference.Domain(imageRef) == reference.Domain(ref) {
		mounter, err := newImageMounter(ctx, resolver, copier, sourceRepoOnly.Name(), repoOnly.Name())
		if err != nil {
			return bundle.BaseImage{}, err
		}
		handler = &mounter
	}

	// Walk the source repository and list all the descriptors
	accumulator := &descriptorAccumulator{}
	if err := images.Walk(ctx, images.Handlers(accumulator, images.ChildrenHandler(&imageContentProvider{sourceFetcher})), descriptor); err != nil {
		return bundle.BaseImage{}, err
	}
	for _, d := range accumulator.descriptors {
		if err := handler.Handle(ctx, d); err != nil {
			return bundle.BaseImage{}, err
		}
	}

	return baseImage, nil
}

func fixupBaseImage(ctx context.Context,
	baseImage *bundle.BaseImage,
	ref opts.NamedOption,
	resolver docker.ResolverBlobMounter) (reference.Named, reference.Named, ocischemav1.Descriptor, error) {
	err := checkBaseImage(baseImage)
	if err != nil {
		err := fmt.Errorf("invalid image %q: %s", ref, err)
		return nil, nil, ocischemav1.Descriptor{}, err
	}
	repoOnly, err := reference.ParseNormalizedNamed(ref.Name())
	if err != nil {
		return nil, nil, ocischemav1.Descriptor{}, err
	}
	imageRef, err := reference.ParseNormalizedNamed(baseImage.Image)
	if err != nil {
		err = fmt.Errorf("%q is not a valid image reference for %q", baseImage.Image, ref)
		return nil, nil, ocischemav1.Descriptor{}, err
	}
	imageRef = reference.TagNameOnly(imageRef)
	_, descriptor, err := resolver.Resolve(ctx, imageRef.String())
	if err != nil {
		err = fmt.Errorf("failed to resolve %q, push the image to the registry before pushing the bundle: %s", imageRef, err)
		return nil, nil, ocischemav1.Descriptor{}, err
	}
	digested, err := reference.WithDigest(repoOnly, descriptor.Digest)
	if err != nil {
		return nil, nil, ocischemav1.Descriptor{}, err
	}
	baseImage.Image = reference.FamiliarString(digested)
	baseImage.MediaType = descriptor.MediaType
	baseImage.Size = uint64(descriptor.Size)
	return repoOnly, imageRef, descriptor, nil
}

func checkBaseImage(baseImage *bundle.BaseImage) error {
	switch baseImage.ImageType {
	case "docker":
	case "oci":
	case "":
		baseImage.ImageType = "oci"
	default:
		return fmt.Errorf("image type %q is not supported", baseImage.ImageType)
	}

	switch baseImage.MediaType {
	case ocischemav1.MediaTypeImageIndex:
	case ocischemav1.MediaTypeImageManifest:
	case images.MediaTypeDockerSchema2Manifest:
	case images.MediaTypeDockerSchema2ManifestList:
	case "":
	default:
		return fmt.Errorf("image media type %q is not supported", baseImage.ImageType)
	}

	return nil
}
