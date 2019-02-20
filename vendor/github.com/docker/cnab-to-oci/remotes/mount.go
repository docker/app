package remotes

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/containerd/containerd/content"
	"github.com/containerd/containerd/errdefs"
	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/remotes"
	"github.com/containerd/containerd/remotes/docker"
	ocischemav1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
)

type imageHandler interface {
	Handle(context.Context, ocischemav1.Descriptor) error
}

func newImageCopier(ctx context.Context, resolver docker.ResolverBlobMounter, sourceFetcher remotes.Fetcher, targetRepo string) (imageCopier, error) {
	destPusher, err := resolver.Pusher(ctx, targetRepo)
	if err != nil {
		return imageCopier{}, err
	}
	return imageCopier{
		sourceFetcher: sourceFetcher,
		targetPusher:  destPusher,
	}, nil
}

type imageCopier struct {
	sourceFetcher remotes.Fetcher
	targetPusher  remotes.Pusher
}

func (h *imageCopier) Handle(ctx context.Context, desc ocischemav1.Descriptor) (err error) {
	if len(desc.URLs) > 0 {
		fmt.Fprintf(os.Stderr, "Skipping foreign descriptor %s with media type %s (size: %d)\n", desc.Digest, desc.MediaType, desc.Size)
		return nil
	}
	fmt.Fprintf(os.Stderr, "Copying descriptor %s with media type %s (size: %d)\n", desc.Digest, desc.MediaType, desc.Size)
	reader, err := h.sourceFetcher.Fetch(ctx, desc)
	if err != nil {
		return err
	}
	defer reader.Close()
	writer, err := h.targetPusher.Push(ctx, desc)
	if err != nil {
		if errors.Cause(err) == errdefs.ErrAlreadyExists {
			return nil
		}
		return err
	}
	defer writer.Close()
	err = content.Copy(ctx, writer, reader, desc.Size, desc.Digest)
	if errors.Cause(err) == errdefs.ErrAlreadyExists {
		return nil
	}
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

func (h *imageMounter) Handle(ctx context.Context, desc ocischemav1.Descriptor) error {
	if len(desc.URLs) > 0 {
		fmt.Fprintf(os.Stderr, "Skipping foreign descriptor %s with media type %s (size: %d)\n", desc.Digest, desc.MediaType, desc.Size)
		return nil
	}
	if isManifest(desc.MediaType) {
		// manifests are copied
		return h.imageCopier.Handle(ctx, desc)
	}
	fmt.Fprintf(os.Stderr, "Mounting descriptor %s with media type %s (size: %d)\n", desc.Digest, desc.MediaType, desc.Size)
	err := h.targetMounter.MountBlob(ctx, desc, h.sourceRepo)
	if errors.Cause(err) == errdefs.ErrAlreadyExists {
		return nil
	}
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

type descriptorAccumulator struct {
	descriptors []ocischemav1.Descriptor
}

func (a *descriptorAccumulator) Handle(ctx context.Context, desc ocischemav1.Descriptor) ([]ocischemav1.Descriptor, error) {
	descs := make([]ocischemav1.Descriptor, len(a.descriptors)+1)
	descs[0] = desc
	for i, d := range a.descriptors {
		descs[i+1] = d
	}

	a.descriptors = descs
	return nil, nil
}
