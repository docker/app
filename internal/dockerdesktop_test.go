package internal

import (
	"testing"

	"github.com/docker/cli/cli/context"
	"github.com/docker/cli/cli/context/docker"
	"github.com/docker/cli/cli/context/kubernetes"
	"github.com/pkg/errors"
	"gotest.tools/assert"
)

var (
	noDesktopProvider = func() (string, bool) {
		return "", false
	}
	desktopProvider = func() (string, bool) {
		return "unix:///test", true
	}
)

func TestDockerDesktopDockerEndpointRewriter(t *testing.T) {
	cases := []struct {
		name         string
		hostProvider dockerDesktopHostProvider
		currentHost  string
		expectedHost string
	}{
		{
			name:         "no-desktop",
			hostProvider: noDesktopProvider,
			currentHost:  "",
			expectedHost: "",
		},
		{
			name:         "no-desktop-custom-host",
			hostProvider: noDesktopProvider,
			currentHost:  "test",
			expectedHost: "test",
		},
		{
			name:         "desktop-empty-host",
			hostProvider: desktopProvider,
			currentHost:  "",
			expectedHost: "unix:///var/run/docker.sock",
		},
		{
			name:         "desktop-default-host",
			hostProvider: desktopProvider,
			currentHost:  "unix:///test",
			expectedHost: "unix:///var/run/docker.sock",
		},
		{
			name:         "desktop-custom-host",
			hostProvider: desktopProvider,
			currentHost:  "test",
			expectedHost: "test",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			testee := dockerDesktopDockerEndpointRewriter{
				defaultHostProvider: c.hostProvider,
			}
			ep := docker.EndpointMeta{
				Host: c.currentHost,
			}
			testee.rewrite(&ep)
			assert.Check(t, ep.Host == c.expectedHost)
		})
	}
}

func TestDockerDesktopKubernetesEndpointRewriter(t *testing.T) {
	cases := []struct {
		name         string
		hostProvider dockerDesktopHostProvider
		ipProvider   dockerDesktopLinuxKitIPProvider
		currentHost  string
		expectedHost string
	}{
		{
			name:         "no-desktop",
			hostProvider: noDesktopProvider,
			currentHost:  "https://localhost:6443",
			expectedHost: "https://localhost:6443",
		},
		{
			name:         "no-desktop-custom-host",
			hostProvider: noDesktopProvider,
			currentHost:  "https://custom:6443",
			expectedHost: "https://custom:6443",
		},
		{
			name:         "desktop-localhost",
			hostProvider: desktopProvider,
			currentHost:  "https://localhost:4242",
			expectedHost: "https://42.42.42.42:6443",
		},
		{
			name:         "desktop-127.0.0.01",
			hostProvider: desktopProvider,
			currentHost:  "https://127.0.0.1:4242",
			expectedHost: "https://42.42.42.42:6443",
		},
		{
			name:         "desktop-custom-host",
			hostProvider: desktopProvider,
			currentHost:  "https://custom:6443",
			expectedHost: "https://custom:6443",
		},
		{
			name:         "no-rewrite-on-error",
			hostProvider: desktopProvider,
			ipProvider: func() (string, error) {
				return "", errors.New("boom")
			},
			currentHost:  "https://127.0.0.1:4242",
			expectedHost: "https://127.0.0.1:4242",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ipProvider := c.ipProvider
			if ipProvider == nil {
				ipProvider = func() (string, error) {
					return "42.42.42.42", nil
				}
			}
			testee := dockerDesktopKubernetesEndpointRewriter{
				defaultHostProvider: c.hostProvider,
				linuxKitIPProvider:  ipProvider,
			}
			ep := kubernetes.EndpointMeta{
				EndpointMetaBase: context.EndpointMetaBase{
					Host: c.currentHost,
				},
			}
			testee.rewrite(&ep)
			assert.Check(t, ep.Host == c.expectedHost)
		})
	}
}
