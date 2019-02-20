package remotes

import (
	"github.com/containerd/containerd/remotes/docker"
	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/docker/registry"
)

// CreateResolver creates a docker registry resolver, using the local docker CLI credentials
func CreateResolver(cfg *configfile.ConfigFile, plainHTTP bool) docker.ResolverBlobMounter {
	return docker.NewResolver(docker.ResolverOptions{
		Authorizer: docker.NewAuthorizer(nil, func(hostName string) (string, string, error) {
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
		}),
		PlainHTTP: plainHTTP,
	})
}
