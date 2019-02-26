package remotes

import (
	"context"
	"fmt"
	"io"

	"github.com/containerd/containerd/content"
	"github.com/containerd/containerd/errdefs"
	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/remotes"
	"github.com/containerd/containerd/remotes/docker"
	ocischemav1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
)

type imageHandler interface {
	Handle(context.Context, *descriptorProgress) error
}

func newImageCopier(ctx context.Context, resolver docker.ResolverBlobMounter, sourceFetcher remotes.Fetcher, targetRepo string, eventNotifier eventNotifier) (imageCopier, error) {
	destPusher, err := resolver.Pusher(ctx, targetRepo)
	if err != nil {
		return imageCopier{}, err
	}
	return imageCopier{
		sourceFetcher: sourceFetcher,
		targetPusher:  destPusher,
		eventNotifier: eventNotifier,
	}, nil
}

type imageCopier struct {
	sourceFetcher remotes.Fetcher
	targetPusher  remotes.Pusher
	eventNotifier eventNotifier
}

func (h *imageCopier) Handle(ctx context.Context, desc *descriptorProgress) (err error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	if len(desc.URLs) > 0 {
		desc.markDone()
		desc.setAction("Skip (foreign layer)")
		h.eventNotifier(FixupEventTypeProgress, "", nil)
		return nil
	}
	desc.setAction("Copy")
	h.eventNotifier(FixupEventTypeProgress, "", nil)
	writer, err := h.targetPusher.Push(ctx, desc.Descriptor)
	if err != nil {
		if errors.Cause(err) == errdefs.ErrAlreadyExists {
			desc.markDone()
			h.eventNotifier(FixupEventTypeProgress, "", nil)
			return nil
		}
		desc.setError(err)
		h.eventNotifier(FixupEventTypeProgress, "", err)
		return err
	}
	defer writer.Close()
	reader, err := h.sourceFetcher.Fetch(ctx, desc.Descriptor)
	if err != nil {
		desc.setError(err)
		h.eventNotifier(FixupEventTypeProgress, "", err)
		return err
	}
	defer reader.Close()
	err = content.Copy(ctx, writer, reader, desc.Size, desc.Digest)
	if errors.Cause(err) == errdefs.ErrAlreadyExists {
		err = nil
	}
	if err != nil {
		desc.setError(err)
	} else {
		desc.markDone()
	}
	h.eventNotifier(FixupEventTypeProgress, "", err)
	return err
}

func newImageMounter(
	ctx context.Context,
	resolver docker.ResolverBlobMounter,
	copier imageCopier,
	sourceRepo,
	targetRepo string) (imageMounter, error) {

	destMounter, err := resolver.BlobMounter(ctx, targetRepo)
	if err != nil {
		return imageMounter{}, err
	}

	return imageMounter{
		imageCopier:   copier,
		targetMounter: destMounter,
		sourceRepo:    sourceRepo,
	}, nil
}

type imageMounter struct {
	imageCopier
	sourceRepo    string
	targetMounter docker.BlobMounter
}

func isManifest(mediaType string) bool {
	return mediaType == images.MediaTypeDockerSchema1Manifest ||
		mediaType == images.MediaTypeDockerSchema2Manifest ||
		mediaType == images.MediaTypeDockerSchema2ManifestList ||
		mediaType == ocischemav1.MediaTypeImageIndex ||
		mediaType == ocischemav1.MediaTypeImageManifest
}

func (h *imageMounter) Handle(ctx context.Context, desc *descriptorProgress) error {
	if len(desc.URLs) > 0 {
		desc.markDone()
		desc.setAction("Skip (foreign layer)")
		h.eventNotifier(FixupEventTypeProgress, "", nil)
		return nil
	}
	if isManifest(desc.MediaType) {
		// manifests are copied
		return h.imageCopier.Handle(ctx, desc)
	}
	desc.setAction("Mount")
	h.eventNotifier(FixupEventTypeProgress, "", nil)
	err := h.targetMounter.MountBlob(ctx, desc.Descriptor, h.sourceRepo)
	if errors.Cause(err) == errdefs.ErrAlreadyExists {
		err = nil
	}
	if err != nil {
		desc.setError(err)
	} else {
		desc.markDone()
	}
	h.eventNotifier(FixupEventTypeProgress, "", err)
	return err
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

type manifestWalker struct {
	getChildren       images.HandlerFunc
	resolver          docker.ResolverBlobMounter
	targetRepo        string
	eventNotifier     eventNotifier
	descriptorHandler imageHandler
	workQueue         *workQueue
	progress          *progress
}

func newManifestWalker(targetRepo string, resolver docker.ResolverBlobMounter, getChildren images.HandlerFunc,
	descriptorHandler imageHandler, eventNotifier eventNotifier, workQueue *workQueue, progress *progress) *manifestWalker {
	return &manifestWalker{
		descriptorHandler: descriptorHandler,
		eventNotifier:     eventNotifier,
		getChildren:       getChildren,
		resolver:          resolver,
		targetRepo:        targetRepo,
		workQueue:         workQueue,
		progress:          progress,
	}
}

var doneCh = func() chan struct{} {
	ch := make(chan struct{})
	close(ch)
	return ch
}()

func (w *manifestWalker) walk(ctx context.Context, desc ocischemav1.Descriptor, parent *descriptorProgress) (chan struct{}, error) {
	descProgress := &descriptorProgress{
		Descriptor: desc,
	}
	if parent != nil {
		parent.addChild(descProgress)
	} else {
		w.progress.addRoot(descProgress)
	}
	copyOrMountWI := func(ctx context.Context) error {
		return w.descriptorHandler.Handle(ctx, descProgress)
	}
	if !isManifest(desc.MediaType) {
		return w.workQueue.enqueue(copyOrMountWI), nil
	}
	_, _, err := w.resolver.Resolve(ctx, fmt.Sprintf("%s@%s", w.targetRepo, desc.Digest))
	if err == nil {
		descProgress.setAction("Skip (already present)")
		descProgress.markDone()
		w.eventNotifier(FixupEventTypeProgress, "", nil)
		return doneCh, nil
	}
	children, err := w.getChildren.Handle(ctx, desc)
	if err != nil {
		return nil, err
	}
	var deps []chan struct{}
	for _, c := range children {
		task, err := w.walk(ctx, c, descProgress)
		if err != nil {
			return nil, err
		}
		deps = append(deps, task)
	}
	return w.workQueue.enqueue(copyOrMountWI, deps...), nil
}

type eventNotifier func(eventType FixupEventType, message string, err error)
