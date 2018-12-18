package commands

import (
	"encoding/json"
	"testing"

	"github.com/deislabs/duffle/pkg/bundle"
	"github.com/docker/app/internal"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/cli/config/types"
	cliflags "github.com/docker/cli/cli/flags"
	"gotest.tools/assert"
)

func TestRequiresBindMount(t *testing.T) {
	dockerCli, err := command.NewDockerCli()
	assert.NilError(t, err)
	dockerCli.Initialize(cliflags.NewClientOptions())

	testCases := []struct {
		name               string
		targetContextName  string
		targetOrchestrator string
		expectedRequired   bool
		expectedEndpoint   string
		expectedError      string
	}{
		{
			name:               "kubernetes-orchestrator",
			targetContextName:  "target-context",
			targetOrchestrator: "kubernetes",
			expectedRequired:   false,
			expectedEndpoint:   "",
			expectedError:      "",
		},
		{
			name:               "no-context",
			targetContextName:  "",
			targetOrchestrator: "swarm",
			expectedRequired:   true,
			expectedEndpoint:   "/var/run/docker.sock",
			expectedError:      "",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result, err := requiredBindMount(testCase.targetContextName, testCase.targetOrchestrator, dockerCli.ContextStore())
			if testCase.expectedError == "" {
				assert.NilError(t, err)
			} else {
				assert.Error(t, err, testCase.expectedError)
			}
			assert.Equal(t, testCase.expectedRequired, result.required)
			assert.Equal(t, testCase.expectedEndpoint, result.endpoint)
		})
	}
}

func TestIsDockerHostLocal(t *testing.T) {
	testCases := []struct {
		name     string
		host     string
		expected bool
	}{
		{
			name:     "not-local",
			host:     "tcp://not.local.host",
			expected: false,
		},
		{
			name:     "no-endpoint",
			host:     "",
			expected: true,
		},
		{
			name:     "docker-sock",
			host:     "unix:///var/run/docker.sock",
			expected: true,
		},
		{
			name:     "named-pipe",
			host:     "npipe:////./pipe/docker_engine",
			expected: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.expected, isDockerHostLocal(testCase.host))
		})
	}
}

func TestSocketPath(t *testing.T) {
	testCases := []struct {
		name     string
		host     string
		expected string
	}{
		{
			name:     "unixSocket",
			host:     "unix:///my/socket.sock",
			expected: "/my/socket.sock",
		},
		{
			name:     "namedPipe",
			host:     "npipe:////./docker",
			expected: "/var/run/docker.sock",
		},
		{
			name:     "emptyHost",
			host:     "",
			expected: "/var/run/docker.sock",
		},
		{
			name:     "tcpHost",
			host:     "tcp://my/tcp/endpoint",
			expected: "/var/run/docker.sock",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.expected, socketPath(testCase.host))
		})
	}
}

type registryConfigMock struct {
	command.Cli
	configFile *configfile.ConfigFile
}

func (r *registryConfigMock) ConfigFile() *configfile.ConfigFile {
	return r.configFile
}

func TestShareRegistryCreds(t *testing.T) {
	cases := []struct {
		name       string
		shareCreds bool
		stored     map[string]types.AuthConfig
		expected   map[string]types.AuthConfig
		images     map[string]bundle.Image
	}{
		{
			name:       "no-share",
			shareCreds: false,
			stored: map[string]types.AuthConfig{
				"my-registry.com": {
					Username: "test",
					Password: "test",
				},
			},
			expected: map[string]types.AuthConfig{},
			images: map[string]bundle.Image{
				"component1": {
					BaseImage: bundle.BaseImage{
						Image: "my-registry.com/ns/repo:tag",
					},
				},
			},
		},
		{
			name:       "share",
			shareCreds: true,
			stored: map[string]types.AuthConfig{
				"my-registry.com": {
					Username: "test",
					Password: "test",
				},
				"my-registry2.com": {
					Username: "test",
					Password: "test",
				},
			},
			expected: map[string]types.AuthConfig{
				"my-registry.com": {
					Username: "test",
					Password: "test",
				}},
			images: map[string]bundle.Image{
				"component1": {
					BaseImage: bundle.BaseImage{
						Image: "my-registry.com/ns/repo:tag",
					},
				},
			},
		},
		{
			name:       "share-missing",
			shareCreds: true,
			stored: map[string]types.AuthConfig{
				"my-registry2.com": {
					Username: "test",
					Password: "test",
				},
			},
			expected: map[string]types.AuthConfig{
				"my-registry.com": {}},
			images: map[string]bundle.Image{
				"component1": {
					BaseImage: bundle.BaseImage{
						Image: "my-registry.com/ns/repo:tag",
					},
				},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			params := map[string]interface{}{
				internal.ParameterShareRegistryCredsName: c.shareCreds,
			}
			creds, err := prepareCredentialSet(
				&bundle.Bundle{
					Credentials: map[string]bundle.Location{internal.CredentialRegistryName: {}},
					Images:      c.images,
				},
				addNamedCredentialSets(nil),
				addDockerCredentials("", nil),
				addRegistryCredentials(params, &registryConfigMock{configFile: &configfile.ConfigFile{
					AuthConfigs: c.stored,
				}}))
			assert.NilError(t, err)
			var result map[string]types.AuthConfig
			assert.NilError(t, json.Unmarshal([]byte(creds[internal.CredentialRegistryName]), &result))
			assert.DeepEqual(t, c.expected, result)
		})
	}
}
