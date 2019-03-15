package commands

import (
	"testing"

	"github.com/docker/cli/cli/command"
	"gotest.tools/assert"
)

func TestRequiresBindMount(t *testing.T) {
	dockerCli, err := command.NewDockerCli()
	assert.NilError(t, err)

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
			result, err := requiredBindMount(testCase.targetContextName, testCase.targetOrchestrator, dockerCli)
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
