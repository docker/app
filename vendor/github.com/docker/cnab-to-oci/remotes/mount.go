package remotes

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/containerd/containerd/content"
	"github.com/containerd/containerd/errdefs"
	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/remotes"
	"github.com/docker/distribution/reference"
	ocischemav1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
)

const (
	// labelDistributionSource describes the source blob comes from.
	// This label comes from containerd: https://github.com/containerd/containerd/blob/master/remotes/docker/handler.go#L35
	labelDistributionSource = "containerd.io/distribution.source"
)

func newDescriptorCopier(ctx context.Context, resolver remotes.Resolver,
	sourceFetcher remotes.Fetcher, targetRepo string,
	eventNotifier eventNotifier, originalSource reference.Named) (*descriptorCopier, error) {
	destPusher, err := resolver.Pusher(ctx, targetRepo)
	if err != nil {
		return nil, err
	}
	return &descriptorCopier{
		sourceFetcher:  sourceFetcher,
		targetPusher:   destPusher,
		eventNotifier:  eventNotifier,
		resolver:       resolver,
		originalSource: originalSource,
	}, nil
}

type descriptorCopier struct {
	sourceFetcher  remotes.Fetcher
	targetPusher   remotes.Pusher
	eventNotifier  eventNotifier
	resolver       remotes.Resolver
	originalSource reference.Named
}

func (h *descriptorCopier) Handle(ctx context.Context, desc *descriptorProgress) (retErr error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	if len(desc.URLs) > 0 {
		desc.markDone()
		desc.setAction("Skip (foreign layer)")
		return nil
	}
	desc.setAction("Copy")
	h.eventNotifier.reportProgress(nil)
	defer func() {
		if retErr != nil {
			desc.setError(retErr)
		}
		h.eventNotifier.reportProgress(retErr)
	}()
	writer, err := pushWithAnnotation(ctx, h.targetPusher, h.originalSource, desc.Descriptor)
	if errors.Cause(err) == errdefs.ErrAlreadyExists {
		desc.markDone()
		if strings.Contains(err.Error(), "mounted") {
			desc.setAction("Mounted")
		}
		return nil
	}
	if err != nil {
		return err
	}
	defer writer.Close()
	reader, err := h.sourceFetcher.Fetch(ctx, desc.Descriptor)
	if err != nil {
		return err
	}
	defer reader.Close()
	err = content.Copy(ctx, writer, reader, desc.Size, desc.Digest)
	if errors.Cause(err) == errdefs.ErrAlreadyExists {
		err = nil
	}
	if err == nil {
		desc.markDone()
	}
	return err
}

func pushWithAnnotation(ctx context.Context, pusher remotes.Pusher, ref reference.Named, desc ocischemav1.Descriptor) (content.Writer, error) {
	// Add the distribution source annotation to help containerd
	// mount instead of push when possible.
	repo := fmt.Sprintf("%s.%s", labelDistributionSource, reference.Domain(ref))
	desc.Annotations = map[string]string{
		repo: reference.FamiliarName(ref),
	}
	return pusher.Push(ctx, desc)
}

func isManifest(mediaType string) bool {
	return mediaType == images.MediaTypeDockerSchema1Manifest ||
		mediaType == images.MediaTypeDockerSchema2Manifest ||
		mediaType == images.MediaTypeDockerSchema2ManifestList ||
		mediaType == ocischemav1.MediaTypeImageIndex ||
		mediaType == ocischemav1.MediaTypeImageManifest
}

type imageContentProvider struct {
	fetcher remotes.Fetcher
}

func (p *imageContentProvider) ReaderAt(ctx context.Context, desc ocischemav1.Descriptor) (content.ReaderAt, error) {
	rc, err := p.fetcher.Fetch(ctx, desc)
	if err != nil {
		return nil, err
	}
	return &remoteReaderAt{ReadCloser: rc, currentOffset: 0, size: desc.Size}, nil
}

type remoteReaderAt struct {
	io.ReadCloser
	currentOffset int64
	size          int64
}

func (r *remoteReaderAt) Size() int64 {
	return r.size
}

func (r *remoteReaderAt) ReadAt(p []byte, off int64) (int, error) {
	if off != r.currentOffset {
		return 0, fmt.Errorf("at the moment this reader only supports offset at %d, requested offset was %d", r.currentOffset, off)
	}
	n, err := r.Read(p)
	r.currentOffset += int64(n)
	if err == io.EOF && n == len(p) {
		return n, nil
	}
	if err != nil || n == len(p) {
		return n, err
	}
	n2, err := r.ReadAt(p[n:], r.currentOffset)
	n += n2
	return n, err
}

type descriptorContentHandler struct {
	descriptorCopier *descriptorCopier
	targetRepo       string
}

func (h *descriptorContentHandler) createCopyTask(ctx context.Context, descProgress *descriptorProgress) (func(ctx context.Context) error, error) {
	copyOrMountWorkItem := func(ctx context.Context) error {
		return h.descriptorCopier.Handle(ctx, descProgress)
	}
	if !isManifest(descProgress.MediaType) {
		return copyOrMountWorkItem, nil
	}
	_, _, err := h.descriptorCopier.resolver.Resolve(ctx, fmt.Sprintf("%s@%s", h.targetRepo, descProgress.Digest))
	if err == nil {
		descProgress.setAction("Skip (already present)")
		descProgress.markDone()
		return nil, errdefs.ErrAlreadyExists
	}
	return copyOrMountWorkItem, nil
}

type manifestWalker struct {
	getChildren    images.HandlerFunc
	eventNotifier  eventNotifier
	scheduler      scheduler
	progress       *progress
	contentHandler *descriptorContentHandler
}

func newManifestWalker(
	eventNotifier eventNotifier,
	scheduler scheduler,
	progress *progress,
	descriptorContentHandler *descriptorContentHandler) *manifestWalker {
	sourceFetcher := descriptorContentHandler.descriptorCopier.sourceFetcher
	return &manifestWalker{
		eventNotifier:  eventNotifier,
		getChildren:    images.ChildrenHandler(&imageContentProvider{sourceFetcher}),
		scheduler:      scheduler,
		progress:       progress,
		contentHandler: descriptorContentHandler,
	}
}

func (w *manifestWalker) walk(ctx context.Context, desc ocischemav1.Descriptor, parent *descriptorProgress) promise {
	select {
	case <-ctx.Done():
		return newPromise(w.scheduler, ctx)
	default:
	}
	descProgress := &descriptorProgress{
		Descriptor: desc,
	}
	if parent != nil {
		parent.addChild(descProgress)
	} else {
		w.progress.addRoot(descProgress)
	}
	copyOrMountWorkItem, err := w.contentHandler.createCopyTask(ctx, descProgress)
	if errors.Cause(err) == errdefs.ErrAlreadyExists {
		w.eventNotifier.reportProgress(nil)
		return newPromise(w.scheduler, doneDependency{})
	}
	if err != nil {
		w.eventNotifier.reportProgress(err)
		return newPromise(w.scheduler, failedDependency{err: err})
	}
	childrenPromise := scheduleAndUnwrap(w.scheduler, func(ctx context.Context) (dependency, error) {
		var deps []dependency
		children, err := w.getChildren.Handle(ctx, desc)
		if err != nil {
			return nil, err
		}
		for _, c := range children {
			dep := w.walk(ctx, c, descProgress)
			deps = append(deps, dep)
		}
		return newPromise(w.scheduler, whenAll(deps)), nil
	})

	return childrenPromise.then(copyOrMountWorkItem)
}

type eventNotifier func(eventType FixupEventType, message string, err error)

func (n eventNotifier) reportProgress(err error) {
	n(FixupEventTypeProgress, "", err)
}
