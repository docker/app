package internal

import (
	"fmt"
	"net/url"
	"runtime"

	"github.com/pkg/errors"

	"github.com/docker/cli/cli/context/docker"
	"github.com/docker/cli/cli/context/kubernetes"
	"github.com/docker/cli/cli/context/store"
	"github.com/docker/docker/client"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

type dockerDesktopHostProvider func() (string, bool)

func defaultDockerDesktopHostProvider() (string, bool) {
	switch runtime.GOOS {
	case "windows", "darwin":
	default:
		// platforms other than windows or mac can't be Docker Desktop
		return "", false
	}
	return client.DefaultDockerHost, true
}

type dockerDesktopLinuxKitIPProvider func() (string, error)

type dockerDesktopDockerEndpointRewriter struct {
	defaultHostProvider dockerDesktopHostProvider
}

func (r *dockerDesktopDockerEndpointRewriter) rewrite(ep *docker.EndpointMeta) {
	defaultHost, isDockerDesktop := r.defaultHostProvider()
	if !isDockerDesktop {
		return
	}
	// on docker desktop, any context with host="" or host=<default host> should be rewritten as host="unix:///var/run/docker.sock" (docker socket path within the linuxkit VM)
	if ep.Host == "" || ep.Host == defaultHost {
		ep.Host = "unix:///var/run/docker.sock"
	}
}

type dockerDesktopKubernetesEndpointRewriter struct {
	defaultHostProvider dockerDesktopHostProvider
	linuxKitIPProvider  dockerDesktopLinuxKitIPProvider
}

func (r *dockerDesktopKubernetesEndpointRewriter) rewrite(ep *kubernetes.EndpointMeta) {
	// any error while rewriting makes as if no rewriting rule applies
	if _, isDockerDesktop := r.defaultHostProvider(); !isDockerDesktop {
		return
	}
	// if the kube endpoint host points to localhost or 127.0.0.1, we need to rewrite it to whatever is linuxkit VM IP is (with port 6443)
	hostURL, err := url.Parse(ep.Host)
	if err != nil {
		return
	}
	hostName := hostURL.Hostname()
	switch hostName {
	case "localhost", "127.0.0.1":
	default:
		// we are on a context targeting a remote Kubernetes cluster, nothing to rewrite
		return
	}
	ip, err := r.linuxKitIPProvider()
	if err != nil {
		return
	}
	ep.Host = fmt.Sprintf("https://%s:6443", ip)
}

// nolint:interfacer
func makeLinuxkitIPProvider(contextName string, s store.Store) dockerDesktopLinuxKitIPProvider {
	return func() (string, error) {
		clientCfg, err := kubernetes.ConfigFromContext(contextName, s)
		if err != nil {
			return "", err
		}
		restCfg, err := clientCfg.ClientConfig()
		if err != nil {
			return "", err
		}
		coreClient, err := v1.NewForConfig(restCfg)
		if err != nil {
			return "", err
		}
		nodes, err := coreClient.Nodes().List(metav1.ListOptions{})
		if err != nil {
			return "", err
		}
		if len(nodes.Items) == 0 {
			return "", errors.New("no node found")
		}
		for _, address := range nodes.Items[0].Status.Addresses {
			if address.Type == apiv1.NodeInternalIP {
				return address.Address, nil
			}
		}
		return "", errors.New("no ip found")
	}
}

func rewriteContextIfDockerDesktop(meta *store.Metadata, s store.Store) {
	// errors are treated as "don't rewrite"
	rewriter := dockerDesktopDockerEndpointRewriter{
		defaultHostProvider: defaultDockerDesktopHostProvider,
	}
	dockerEp, err := docker.EndpointFromContext(*meta)
	if err != nil {
		return
	}
	rewriter.rewrite(&dockerEp)
	meta.Endpoints[docker.DockerEndpoint] = dockerEp
	kubeEp := kubernetes.EndpointFromContext(*meta)
	if kubeEp == nil {
		return
	}
	kubeRewriter := dockerDesktopKubernetesEndpointRewriter{
		defaultHostProvider: defaultDockerDesktopHostProvider,
		linuxKitIPProvider:  makeLinuxkitIPProvider(meta.Name, s),
	}
	kubeRewriter.rewrite(kubeEp)
	meta.Endpoints[kubernetes.KubernetesEndpoint] = *kubeEp
}

type DockerDesktopAwareStore struct {
	store.Store
}

func (s DockerDesktopAwareStore) List() ([]store.Metadata, error) {
	contexts, err := s.Store.List()
	if err != nil {
		return nil, err
	}
	for ix, c := range contexts {
		rewriteContextIfDockerDesktop(&c, s.Store)
		contexts[ix] = c
	}
	return contexts, nil
}

func (s DockerDesktopAwareStore) GetMetadata(name string) (store.Metadata, error) {
	context, err := s.Store.GetMetadata(name)
	if err != nil {
		return store.Metadata{}, err
	}
	rewriteContextIfDockerDesktop(&context, s.Store)
	return context, nil
}
