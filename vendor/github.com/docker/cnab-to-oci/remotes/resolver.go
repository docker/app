package remotes

import (
	"context"

	"github.com/containerd/containerd/remotes"
	"github.com/containerd/containerd/remotes/docker"
	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/distribution/reference"
	"github.com/docker/docker/registry"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

type multiRegistryResolver struct {
	plainHTTP           docker.ResolverBlobMounter
	secure              docker.ResolverBlobMounter
	plainHTTPRegistries map[string]struct{}
}

func (r *multiRegistryResolver) resolveImplementation(image string) (docker.ResolverBlobMounter, error) {
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
	return r.secure, nil
}

func (r *multiRegistryResolver) BlobMounter(ctx context.Context, ref string) (docker.BlobMounter, error) {
	impl, err := r.resolveImplementation(ref)
	if err != nil {
		return nil, err
	}
	return impl.BlobMounter(ctx, ref)
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
func CreateResolver(cfg *configfile.ConfigFile, plainHTTPRegistries ...string) docker.ResolverBlobMounter {
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

	result := &multiRegistryResolver{
		plainHTTP: docker.NewResolver(docker.ResolverOptions{
			Authorizer: authorizer,
			PlainHTTP:  true,
		}),
		secure: docker.NewResolver(docker.ResolverOptions{
			Authorizer: authorizer,
			PlainHTTP:  false,
		}),
		plainHTTPRegistries: make(map[string]struct{}),
	}

	for _, r := range plainHTTPRegistries {
		result.plainHTTPRegistries[r] = struct{}{}
	}

	return result
}
