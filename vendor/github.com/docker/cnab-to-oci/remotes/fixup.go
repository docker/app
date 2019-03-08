package remotes

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/remotes"
	"github.com/deislabs/duffle/pkg/bundle"
	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/distribution/reference"
	ocischemav1 "github.com/opencontainers/image-spec/specs-go/v1"
)

const (
	defaultMaxConcurrentJobs = 4
	defaultJobsBufferLength  = 50
)

func noopEventCallback(FixupEvent) {}

// fixupConfig defines the input required for a Fixup operation
type fixupConfig struct {
	bundle            *bundle.Bundle
	targetRef         reference.Named
	eventCallback     func(FixupEvent)
	maxConcurrentJobs int
	jobsBufferLength  int
	resolverConfig    ResolverConfig
}

func (cfg *fixupConfig) complete() error {
	if cfg.resolverConfig.Resolver == nil || cfg.resolverConfig.OriginProviderWrapper == nil {
		return errors.New("resolver and originProviderWrapper are required, please use a complete ResolverConfig")
	}
	return nil
}

// WithEventCallback specifies a callback to execute for each Fixup event
func WithEventCallback(callback func(FixupEvent)) FixupOption {
	return func(cfg *fixupConfig) error {
		cfg.eventCallback = callback
		return nil
	}
}

// WithParallelism provides a way to change the max concurrent jobs and the max number of jobs queued up
func WithParallelism(maxConcurrentJobs int, jobsBufferLength int) FixupOption {
	return func(cfg *fixupConfig) error {
		cfg.maxConcurrentJobs = maxConcurrentJobs
		cfg.jobsBufferLength = jobsBufferLength
		return nil
	}
}

// FixupOption is a helper for configuring a FixupBundle
type FixupOption func(*fixupConfig) error

// ResolverConfig represents a resolver and its associated OriginProviderWrapper
type ResolverConfig struct {
	Resolver              remotes.Resolver
	OriginProviderWrapper OriginProviderWrapper
}

// NewResolverConfig creates a ResolverConfig
func NewResolverConfig(resolver remotes.Resolver, originProviderWrapper OriginProviderWrapper) ResolverConfig {
	return ResolverConfig{
		Resolver:              resolver,
		OriginProviderWrapper: originProviderWrapper,
	}
}

// NewResolverConfigFromDockerConfigFile creates a ResolverConfig from a docker CLI config file and a list of registries to reach
// using plain HTTP
func NewResolverConfigFromDockerConfigFile(cfg *configfile.ConfigFile, plainHTTPRegistries ...string) ResolverConfig {
	resolver, originProviderWrapper := CreateResolver(cfg, plainHTTPRegistries...)
	return NewResolverConfig(resolver, originProviderWrapper)
}

func newFixupConfig(b *bundle.Bundle, ref reference.Named, resolverConfig ResolverConfig, options ...FixupOption) (fixupConfig, error) {
	cfg := fixupConfig{
		bundle:            b,
		targetRef:         ref,
		resolverConfig:    resolverConfig,
		eventCallback:     noopEventCallback,
		jobsBufferLength:  defaultJobsBufferLength,
		maxConcurrentJobs: defaultMaxConcurrentJobs,
	}
	for _, opt := range options {
		if err := opt(&cfg); err != nil {
			return fixupConfig{}, err
		}
	}
	if err := cfg.complete(); err != nil {
		return fixupConfig{}, err
	}
	return cfg, nil
}

// FixupBundle checks that all the references are present in the referenced repository, otherwise it will mount all
// the manifests to that repository. The bundle is then patched with the new digested references.
func FixupBundle(ctx context.Context, b *bundle.Bundle, ref reference.Named, resolverConfig ResolverConfig, opts ...FixupOption) error {
	cfg, err := newFixupConfig(b, ref, resolverConfig, opts...)
	if err != nil {
		return err
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

	if len(b.InvocationImages) != 1 {
		return fmt.Errorf("only one invocation image supported for bundle %q", ref)
	}
	if b.InvocationImages[0].BaseImage, err = fixupImage(ctx, b.InvocationImages[0].BaseImage, cfg, events); err != nil {
		return err
	}
	for name, original := range b.Images {
		if original.BaseImage, err = fixupImage(ctx, original.BaseImage, cfg, events); err != nil {
			return err
		}
		b.Images[name] = original
	}
	return nil
}

func fixupImage(ctx context.Context, baseImage bundle.BaseImage, cfg fixupConfig, events chan<- FixupEvent) (_ bundle.BaseImage, retErr error) {
	progress := &progress{}
	originalSource := baseImage.Image
	notifyEvent := func(eventType FixupEventType, message string, err error) {
		events <- FixupEvent{
			DestinationRef: cfg.targetRef,
			SourceImage:    originalSource,
			EventType:      eventType,
			Message:        message,
			Error:          err,
			Progress:       progress.snapshot(),
		}
	}
	defer func() {
		if retErr != nil {
			notifyEvent(FixupEventTypeCopyImageEnd, "", retErr)
		}
	}()
	notifyEvent(FixupEventTypeCopyImageStart, "", nil)
	repoOnly, imageRef, descriptor, err := fixupBaseImage(ctx, &baseImage, cfg.targetRef, cfg.resolverConfig.Resolver)
	if err != nil {
		return bundle.BaseImage{}, err
	}
	if imageRef.Name() == cfg.targetRef.Name() {
		notifyEvent(FixupEventTypeCopyImageEnd, "Nothing to do: image reference is already present in repository"+repoOnly.String(), nil)
		return baseImage, nil
	}
	sourceRepoOnly, err := reference.ParseNormalizedNamed(imageRef.Name())
	if err != nil {
		return bundle.BaseImage{}, err
	}
	sourceFetcher, err := cfg.resolverConfig.Resolver.Fetcher(ctx, sourceRepoOnly.Name())
	if err != nil {
		return bundle.BaseImage{}, err
	}
	if err := setFromImageReference(cfg.resolverConfig.OriginProviderWrapper, imageRef); err != nil {
		return bundle.BaseImage{}, err
	}

	// Prepare the copier
	copier, err := newDescriptorCopier(ctx, cfg.resolverConfig.Resolver, sourceFetcher, repoOnly.String(), notifyEvent)
	if err != nil {
		return bundle.BaseImage{}, err
	}
	descriptorContentHandler := &descriptorContentHandler{
		descriptorCopier: copier,
		targetRepo:       repoOnly.String(),
	}
	ctx, cancel := context.WithCancel(ctx)
	scheduler := newErrgroupScheduler(ctx, cfg.maxConcurrentJobs, cfg.jobsBufferLength)
	defer func() {
		cancel()
		scheduler.drain()
	}()
	walker := newManifestWalker(notifyEvent, scheduler, progress, descriptorContentHandler)
	walkerDep := walker.walk(scheduler.ctx(), descriptor, nil)
	if err = walkerDep.wait(); err != nil {
		return bundle.BaseImage{}, err
	}
	notifyEvent(FixupEventTypeCopyImageEnd, "", nil)
	return baseImage, nil
}

func fixupBaseImage(ctx context.Context,
	baseImage *bundle.BaseImage,
	ref reference.Named, //nolint: interfacer
	resolver remotes.Resolver) (reference.Named, reference.Named, ocischemav1.Descriptor, error) {
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
	mut      sync.RWMutex
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
	p.mut.RLock()
	defer p.mut.RUnlock()
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
	roots []*descriptorProgress
	mut   sync.RWMutex
}

func (p *progress) addRoot(root *descriptorProgress) {
	p.mut.Lock()
	defer p.mut.Unlock()
	p.roots = append(p.roots, root)
}

func (p *progress) snapshot() ProgressSnapshot {
	p.mut.RLock()
	defer p.mut.RUnlock()
	result := ProgressSnapshot{}
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
	Roots []DescriptorProgressSnapshot
}
