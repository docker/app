package inspect

import (
	"testing"

	composetypes "github.com/docker/cli/cli/compose/types"
	"gotest.tools/assert"
)

func TestGetPorts(t *testing.T) {
	for _, testcase := range []struct {
		name     string
		ports    []composetypes.ServicePortConfig
		expected string
	}{
		{
			name:     "no-published-ports",
			ports:    []composetypes.ServicePortConfig{target(80)},
			expected: "",
		},
		{
			name:     "published-port",
			ports:    []composetypes.ServicePortConfig{published(8080)},
			expected: "8080",
		},
		{
			name:     "mix-published-target-ports",
			ports:    []composetypes.ServicePortConfig{published(8080), target(80), published(9090)},
			expected: "8080,9090",
		},
		{
			name:     "simple-range",
			ports:    publishedRange(8080, 8085),
			expected: "8080-8085",
		},
		{
			name:     "complex-range",
			ports:    append(append(publishedRange(8080, 8081), target(80), published(8082)), publishedRange(8083, 8090)...),
			expected: "8080-8090",
		},
		{
			name:     "multi-range",
			ports:    append(append(publishedRange(8080, 8081), published(8083)), publishedRange(8085, 8086)...),
			expected: "8080-8081,8083,8085-8086",
		},
		{
			name:     "ports-are-sorted",
			ports:    []composetypes.ServicePortConfig{published(8080), published(8082), published(7979), published(8081)},
			expected: "7979,8080-8082",
		},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			assert.Equal(t, getPorts(testcase.ports), testcase.expected)
		})
	}
}

func published(port uint32) composetypes.ServicePortConfig {
	return composetypes.ServicePortConfig{
		Published: port,
	}
}

func target(port uint32) composetypes.ServicePortConfig {
	return composetypes.ServicePortConfig{
		Target: port,
	}
}

func publishedRange(start, end uint32) []composetypes.ServicePortConfig {
	var ports []composetypes.ServicePortConfig
	for i := start; i <= end; i++ {
		ports = append(ports, published(i))
	}
	return ports
}
