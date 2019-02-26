package remotes

import (
	"context"
	"fmt"
	"sync"

	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/remotes/docker"
	"github.com/deislabs/duffle/pkg/bundle"
	"github.com/docker/distribution/reference"
	ocischemav1 "github.com/opencontainers/image-spec/specs-go/v1"
)

// FixupBundle checks that all the references are present in the referenced repository, otherwise it will mount all
// the manifests to that repository. The bundle is then patched with the new digested references.
func FixupBundle(ctx context.Context, b *bundle.Bundle, ref reference.Named, resolver docker.ResolverBlobMounter, events chan<- FixupEvent) error {
	if len(b.InvocationImages) != 1 {
		return fmt.Errorf("only one invocation image supported for bundle %q", ref)
	}
	var err error
	if b.InvocationImages[0].BaseImage, err = fixupImage(ctx, b.InvocationImages[0].BaseImage, ref, resolver, events); err != nil {
		return err
	}
	for name, original := range b.Images {
		if original.BaseImage, err = fixupImage(ctx, original.BaseImage, ref, resolver, events); err != nil {
			return err
		}
		b.Images[name] = original
	}
	return nil
}

func fixupImage(ctx context.Context, baseImage bundle.BaseImage, ref reference.Named, resolver docker.ResolverBlobMounter, events chan<- FixupEvent) (bundle.BaseImage, error) {
	progress := &progress{}
	events <- FixupEvent{
		DestinationRef: ref,
		SourceImage:    baseImage.Image,
		EventType:      FixupEventTypeCopyImageStart,
		Progress:       progress.snapshot(),
	}
	originalSource := baseImage.Image
	notifyEvent := func(eventType FixupEventType, message string, err error) {

		events <- FixupEvent{
			DestinationRef: ref,
			SourceImage:    originalSource,
			EventType:      eventType,
			Message:        message,
			Error:          err,
			Progress:       progress.snapshot(),
		}
	}
	notifyImageEndError := func(err error) {
		notifyEvent(FixupEventTypeCopyImageEnd, "", err)
	}
	repoOnly, imageRef, descriptor, err := fixupBaseImage(ctx, &baseImage, ref, resolver)
	if err != nil {
		notifyImageEndError(err)
		return bundle.BaseImage{}, err
	}
	if imageRef.Name() == ref.Name() {
		events <- FixupEvent{
			DestinationRef: ref,
			SourceImage:    originalSource,
			EventType:      FixupEventTypeCopyImageEnd,
			Message:        "Nothing to do: image reference is in the same repository than " + ref.String(),
		}
		return baseImage, nil
	}
	sourceRepoOnly, err := reference.ParseNormalizedNamed(imageRef.Name())
	if err != nil {
		notifyImageEndError(err)
		return bundle.BaseImage{}, err
	}
	sourceFetcher, err := resolver.Fetcher(ctx, sourceRepoOnly.Name())
	if err != nil {
		notifyImageEndError(err)
		return bundle.BaseImage{}, err
	}

	// Prepare the copier or the mounter
	copier, err := newImageCopier(ctx, resolver, sourceFetcher, repoOnly.String(), notifyEvent)
	if err != nil {
		notifyImageEndError(err)
		return bundle.BaseImage{}, err
	}
	var handler imageHandler = &copier
	if reference.Domain(imageRef) == reference.Domain(ref) {
		mounter, err := newImageMounter(ctx, resolver, copier, sourceRepoOnly.Name(), repoOnly.Name())
		if err != nil {
			notifyImageEndError(err)
			return bundle.BaseImage{}, err
		}
		handler = &mounter
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	wq := newWorkQueue(ctx, 4, 10)
	walker := newManifestWalker(repoOnly.String(), resolver, images.ChildrenHandler(&imageContentProvider{sourceFetcher}),
		handler, notifyEvent, wq, progress)
	if _, err = walker.walk(ctx, descriptor, nil); err != nil {
		notifyImageEndError(err)
		wq.stopAndWait()
		return bundle.BaseImage{}, err
	}
	progress.markWalkDone()
	notifyEvent(FixupEventTypeProgress, "", nil)
	if err = wq.stopAndWait(); err != nil {
		notifyImageEndError(err)
		return bundle.BaseImage{}, err
	}

	events <- FixupEvent{
		DestinationRef: ref,
		SourceImage:    originalSource,
		EventType:      FixupEventTypeCopyImageEnd,
	}
	return baseImage, nil
}

func fixupBaseImage(ctx context.Context,
	baseImage *bundle.BaseImage,
	ref reference.Named,
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

// FixupEvent is an event that is raised by the Fixup Logic
type FixupEvent struct {
	SourceImage    string
	DestinationRef reference.Named
	EventType      FixupEventType
	Message        string
	Error          error
	Progress       ProgressSnapshot
}

// FixupEventType is the the type of event raised by the Fixup logic
type FixupEventType string

const (
	// FixupEventTypeCopyImageStart is raised when the Fixup logic starts copying an
	// image
	FixupEventTypeCopyImageStart = FixupEventType("CopyImageStart")

	// FixupEventTypeCopyImageEnd is raised when the Fixup logic stops copying an
	// image. Error might be populated
	FixupEventTypeCopyImageEnd = FixupEventType("CopyImageEnd")

	// FixupEventTypeProgress is raised when Fixup logic reports progression
	FixupEventTypeProgress = FixupEventType("Progress")
)

type descriptorProgress struct {
	ocischemav1.Descriptor
	done     bool
	action   string
	err      error
	children []*descriptorProgress
	mut      sync.Mutex
}

func (p *descriptorProgress) markDone() {
	p.mut.Lock()
	defer p.mut.Unlock()
	p.done = true
}

func (p *descriptorProgress) setAction(a string) {
	p.mut.Lock()
	defer p.mut.Unlock()
	p.action = a
}

func (p *descriptorProgress) setError(err error) {
	p.mut.Lock()
	defer p.mut.Unlock()
	p.err = err
}

func (p *descriptorProgress) addChild(child *descriptorProgress) {
	p.mut.Lock()
	defer p.mut.Unlock()
	p.children = append(p.children, child)
}

func (p *descriptorProgress) snapshot() DescriptorProgressSnapshot {
	p.mut.Lock()
	defer p.mut.Unlock()
	result := DescriptorProgressSnapshot{
		Descriptor: p.Descriptor,
		Done:       p.done,
		Action:     p.action,
		Error:      p.err,
	}
	if len(p.children) != 0 {
		result.Children = make([]DescriptorProgressSnapshot, len(p.children))
		for ix, child := range p.children {
			result.Children[ix] = child.snapshot()
		}
	}
	return result
}

type progress struct {
	roots    []*descriptorProgress
	walkDone bool
	mut      sync.Mutex
}

func (p *progress) addRoot(root *descriptorProgress) {
	p.mut.Lock()
	defer p.mut.Unlock()
	p.roots = append(p.roots, root)
}

func (p *progress) markWalkDone() {
	p.mut.Lock()
	defer p.mut.Unlock()
	p.walkDone = true
}

func (p *progress) snapshot() ProgressSnapshot {
	p.mut.Lock()
	defer p.mut.Unlock()
	result := ProgressSnapshot{
		WalkDone: p.walkDone,
	}
	if len(p.roots) != 0 {
		result.Roots = make([]DescriptorProgressSnapshot, len(p.roots))
		for ix, root := range p.roots {
			result.Roots[ix] = root.snapshot()
		}
	}
	return result
}

// DescriptorProgressSnapshot describes the current progress of a descriptor
type DescriptorProgressSnapshot struct {
	ocischemav1.Descriptor
	Done     bool
	Action   string
	Error    error
	Children []DescriptorProgressSnapshot
}

// ProgressSnapshot describes the current progress of a Fixup operation
type ProgressSnapshot struct {
	Roots    []DescriptorProgressSnapshot
	WalkDone bool
}
