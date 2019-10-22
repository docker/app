package commands

import (
	"testing"

	"gotest.tools/assert"
)

func TestPlatformFilter(t *testing.T) {
	cases := []struct {
		name     string
		opts     pushOptions
		expected []string
	}{
		{
			name: "filtered-platforms",
			opts: pushOptions{
				allPlatforms: false,
				platforms:    []string{"linux/amd64", "linux/arm64"},
			},
			expected: []string{"linux/amd64", "linux/arm64"},
		},
		{
			name: "all-platforms",
			opts: pushOptions{
				allPlatforms: true,
				platforms:    []string{"linux/amd64"},
			},
			expected: nil,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.DeepEqual(t, platformFilter(c.opts), c.expected)
		})
	}
}
