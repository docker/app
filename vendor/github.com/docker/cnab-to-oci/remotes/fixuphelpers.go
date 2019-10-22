package remotes

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/remotes"
	"github.com/deislabs/cnab-go/bundle"
	"github.com/docker/distribution/reference"
	"github.com/opencontainers/go-digest"
	ocischemav1 "github.com/opencontainers/image-spec/specs-go/v1"
)

type sourceFetcherAdder interface {
	remotes.Fetcher
	Add(data []byte) digest.Digest
}

type sourceFetcherWithLocalData struct {
	inner     remotes.Fetcher
	localData map[digest.Digest][]byte
}

func newSourceFetcherWithLocalData(inner remotes.Fetcher) *sourceFetcherWithLocalData {
	return &sourceFetcherWithLocalData{
		inner:     inner,
		localData: make(map[digest.Digest][]byte),
	}
}

func (s *sourceFetcherWithLocalData) Add(data []byte) digest.Digest {
	d := digest.FromBytes(data)
	s.localData[d] = data
	return d
}

func (s *sourceFetcherWithLocalData) Fetch(ctx context.Context, desc ocischemav1.Descriptor) (io.ReadCloser, error) {
	if v, ok := s.localData[desc.Digest]; ok {
		return ioutil.NopCloser(bytes.NewReader(v)), nil
	}
	return s.inner.Fetch(ctx, desc)
}

type imageFixupInfo struct {
	targetRepo         reference.Named
	sourceRef          reference.Named
	resolvedDescriptor ocischemav1.Descriptor
}

func makeEventNotifier(events chan<- FixupEvent, baseImage string, targetRef reference.Named) (eventNotifier, *progress) {
	progress := &progress{}
	return func(eventType FixupEventType, message string, err error) {
		events <- FixupEvent{
			DestinationRef: targetRef,
			SourceImage:    baseImage,
			EventType:      eventType,
			Message:        message,
			Error:          err,
			Progress:       progress.snapshot(),
		}
	}, progress
}

func makeSourceFetcher(ctx context.Context, resolver remotes.Resolver, sourceRef string) (*sourceFetcherWithLocalData, error) {
	sourceRepoOnly, err := reference.ParseNormalizedNamed(sourceRef)
	if err != nil {
		return nil, err
	}
	f, err := resolver.Fetcher(ctx, sourceRepoOnly.Name())
	if err != nil {
		return nil, err
	}
	return newSourceFetcherWithLocalData(f), nil
}

func makeManifestWalker(ctx context.Context, sourceFetcher remotes.Fetcher,
	notifyEvent eventNotifier, cfg fixupConfig, fixupInfo imageFixupInfo, progress *progress) (promise, func(), error) {
	copier, err := newDescriptorCopier(ctx, cfg.resolver, sourceFetcher, fixupInfo.targetRepo.String(), notifyEvent, fixupInfo.sourceRef)
	if err != nil {
		return promise{}, nil, err
	}
	descriptorContentHandler := &descriptorContentHandler{
		descriptorCopier: copier,
		targetRepo:       fixupInfo.targetRepo.String(),
	}
	ctx, cancel := context.WithCancel(ctx)
	scheduler := newErrgroupScheduler(ctx, cfg.maxConcurrentJobs, cfg.jobsBufferLength)
	cleaner := func() {
		cancel()
		scheduler.drain() //nolint:errcheck
	}
	walker := newManifestWalker(notifyEvent, scheduler, progress, descriptorContentHandler)
	return walker.walk(scheduler.ctx(), fixupInfo.resolvedDescriptor, nil), cleaner, nil
}

func notifyError(notifyEvent eventNotifier, err error) error {
	notifyEvent(FixupEventTypeCopyImageEnd, "", err)
	return err
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
		return fmt.Errorf("image media type %q is not supported", baseImage.MediaType)
	}

	return nil
}
