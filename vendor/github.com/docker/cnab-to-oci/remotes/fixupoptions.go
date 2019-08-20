package remotes

import (
	"github.com/containerd/containerd/platforms"
	"github.com/containerd/containerd/remotes"
	"github.com/deislabs/cnab-go/bundle"
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
	bundle                        *bundle.Bundle
	targetRef                     reference.Named
	eventCallback                 func(FixupEvent)
	maxConcurrentJobs             int
	jobsBufferLength              int
	resolver                      remotes.Resolver
	invocationImagePlatformFilter platforms.Matcher
	componentImagePlatformFilter  platforms.Matcher
}

// FixupOption is a helper for configuring a FixupBundle
type FixupOption func(*fixupConfig) error

func newFixupConfig(b *bundle.Bundle, ref reference.Named, resolver remotes.Resolver, options ...FixupOption) (fixupConfig, error) {
	cfg := fixupConfig{
		bundle:            b,
		targetRef:         ref,
		resolver:          resolver,
		eventCallback:     noopEventCallback,
		jobsBufferLength:  defaultJobsBufferLength,
		maxConcurrentJobs: defaultMaxConcurrentJobs,
	}
	for _, opt := range options {
		if err := opt(&cfg); err != nil {
			return fixupConfig{}, err
		}
	}
	return cfg, nil
}

// WithInvocationImagePlatforms use filters platforms for an invocation image
func WithInvocationImagePlatforms(supportedPlatforms []string) FixupOption {
	return func(cfg *fixupConfig) error {
		if len(supportedPlatforms) == 0 {
			return nil
		}
		plats, err := toPlatforms(supportedPlatforms)
		if err != nil {
			return err
		}
		cfg.invocationImagePlatformFilter = platforms.Any(plats...)
		return nil
	}
}

// WithComponentImagePlatforms use filters platforms for an invocation image
func WithComponentImagePlatforms(supportedPlatforms []string) FixupOption {
	return func(cfg *fixupConfig) error {
		if len(supportedPlatforms) == 0 {
			return nil
		}
		plats, err := toPlatforms(supportedPlatforms)
		if err != nil {
			return err
		}
		cfg.componentImagePlatformFilter = platforms.Any(plats...)
		return nil
	}
}

func toPlatforms(supportedPlatforms []string) ([]ocischemav1.Platform, error) {
	result := make([]ocischemav1.Platform, len(supportedPlatforms))
	for ix, p := range supportedPlatforms {
		plat, err := platforms.Parse(p)
		if err != nil {
			return nil, err
		}
		result[ix] = plat
	}
	return result, nil
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
