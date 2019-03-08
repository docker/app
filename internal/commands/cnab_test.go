package commands

import (
	"testing"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/context"
	"gotest.tools/assert"
)

func TestRequiresBindMount(t *testing.T) {
	dockerCli, err := command.NewDockerCli()
	assert.NilError(t, err)

	testCases := []struct {
		name               string
		targetContextName  string
		targetOrchestrator string
		expectedResult     bool
		expectedError      string
	}{
		{
			name:               "kubernetes-orchestrator",
			targetContextName:  "target-context",
			targetOrchestrator: "kubernetes",
			expectedResult:     false,
			expectedError:      "",
		},
		{
			name:               "no-context",
			targetContextName:  "",
			targetOrchestrator: "swarm",
			expectedResult:     true,
			expectedError:      "",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result, err := requiresBindMount(testCase.targetContextName, testCase.targetOrchestrator, dockerCli)
			if testCase.expectedError == "" {
				assert.NilError(t, err)
			} else {
				assert.Error(t, err, testCase.expectedError)
			}
			assert.Equal(t, testCase.expectedResult, result)
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
			dockerEndpoint := context.EndpointMetaBase{Host: testCase.host}
			assert.Equal(t, testCase.expected, isDockerHostLocal(dockerEndpoint))
		})
	}
}
