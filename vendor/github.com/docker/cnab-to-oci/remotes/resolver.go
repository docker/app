package remotes

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"

	containerdreference "github.com/containerd/containerd/reference"
	"github.com/containerd/containerd/remotes"
	"github.com/containerd/containerd/remotes/docker"
	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/distribution/reference"
	"github.com/docker/docker/registry"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

type multiRegistryResolver struct {
	plainHTTP           remotes.Resolver
	secure              remotes.Resolver
	skipTLS             remotes.Resolver
	plainHTTPRegistries map[string]struct{}
	skipTLSRegistries   map[string]struct{}
}

func (r *multiRegistryResolver) resolveImplementation(image string) (remotes.Resolver, error) {
	ref, err := reference.ParseNormalizedNamed(image)
	if err != nil {
		return nil, err
	}
	repoInfo, err := registry.ParseRepositoryInfo(ref)
	if err != nil {
		return nil, err
	}
	if _, plainHTTP := r.plainHTTPRegistries[repoInfo.Index.Name]; plainHTTP {
		return r.plainHTTP, nil
	}
	if _, skipTLS := r.skipTLSRegistries[repoInfo.Index.Name]; skipTLS {
		return r.skipTLS, nil
	}
	return r.secure, nil
}

func (r *multiRegistryResolver) Resolve(ctx context.Context, ref string) (name string, desc ocispec.Descriptor, err error) {
	impl, err := r.resolveImplementation(ref)
	if err != nil {
		return "", ocispec.Descriptor{}, err
	}
	return impl.Resolve(ctx, ref)
}

func (r *multiRegistryResolver) Fetcher(ctx context.Context, ref string) (remotes.Fetcher, error) {
	impl, err := r.resolveImplementation(ref)
	if err != nil {
		return nil, err
	}
	return impl.Fetcher(ctx, ref)
}

func (r *multiRegistryResolver) Pusher(ctx context.Context, ref string) (remotes.Pusher, error) {
	impl, err := r.resolveImplementation(ref)
	if err != nil {
		return nil, err
	}
	return impl.Pusher(ctx, ref)
}

// CreateResolver creates a docker registry resolver, using the local docker CLI credentials
func CreateResolver(cfg *configfile.ConfigFile, plainHTTPRegistries ...string) (remotes.Resolver, OriginProviderWrapper) {
	authorizer := docker.NewAuthorizer(nil, func(hostName string) (string, string, error) {
		if hostName == registry.DefaultV2Registry.Host {
			hostName = registry.IndexServer
		}
		a, err := cfg.GetAuthConfig(hostName)
		if err != nil {
			return "", "", err
		}
		if a.IdentityToken != "" {
			return "", a.IdentityToken, nil
		}
		return a.Username, a.Password, nil
	})

	clientSkipTLS := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	skipTLSAuthorizer := docker.NewAuthorizer(clientSkipTLS, func(hostName string) (string, string, error) {
		if hostName == registry.DefaultV2Registry.Host {
			hostName = registry.IndexServer
		}
		a, err := cfg.GetAuthConfig(hostName)
		if err != nil {
			return "", "", err
		}
		if a.IdentityToken != "" {
			return "", a.IdentityToken, nil
		}
		return a.Username, a.Password, nil
	})

	originProvider := &originProviderWrapper{}

	result := &multiRegistryResolver{
		plainHTTP: docker.NewResolver(docker.ResolverOptions{
			Authorizer:     authorizer,
			PlainHTTP:      true,
			OriginProvider: originProvider.resolveSource,
		}),
		secure: docker.NewResolver(docker.ResolverOptions{
			Authorizer:     authorizer,
			PlainHTTP:      false,
			OriginProvider: originProvider.resolveSource,
		}),
		skipTLS: docker.NewResolver(docker.ResolverOptions{
			Authorizer:     skipTLSAuthorizer,
			PlainHTTP:      false,
			OriginProvider: originProvider.resolveSource,
			Client:         clientSkipTLS,
		}),
		plainHTTPRegistries: make(map[string]struct{}),
		skipTLSRegistries:   make(map[string]struct{}),
	}

	for _, r := range plainHTTPRegistries {
		pingURL := fmt.Sprintf("https://%s/v2/", r)
		resp, err := clientSkipTLS.Get(pingURL)
		if err == nil {
			resp.Body.Close()
			result.skipTLSRegistries[r] = struct{}{}
		} else {
			result.plainHTTPRegistries[r] = struct{}{}
		}
	}

	return result, originProvider
}

// OriginProviderWrapper wraps an origin provider, to be able to change the origin provider implementation
// after having created the resolver
type OriginProviderWrapper interface {
	Wrap(func(ocispec.Descriptor) []containerdreference.Spec)
}

type originProviderWrapper struct {
	originProvider func(ocispec.Descriptor) []containerdreference.Spec
}

func (p *originProviderWrapper) resolveSource(desc ocispec.Descriptor) []containerdreference.Spec {
	if p.originProvider == nil {
		return nil
	}
	return p.originProvider(desc)
}

func (p *originProviderWrapper) Wrap(provider func(ocispec.Descriptor) []containerdreference.Spec) {
	p.originProvider = provider
}

// nolint: interfacer
func setFromImageReference(wrapper OriginProviderWrapper, named reference.Named) error {
	spec, err := containerdreference.Parse(named.Name())
	if err != nil {
		return err
	}
	origins := []containerdreference.Spec{spec}
	wrapper.Wrap(func(_ ocispec.Descriptor) []containerdreference.Spec {
		return origins
	})
	return nil
}
