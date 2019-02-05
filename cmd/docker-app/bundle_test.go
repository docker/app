package main

import (
	"testing"

	"github.com/docker/app/types/metadata"
	"gotest.tools/assert"
)

func TestMakeInvocationImage(t *testing.T) {
	testcases := []struct {
		name      string
		imageName string
		meta      metadata.AppMetadata
		expected  string
		err       string
	}{
		{
			name:      "specify-image-name",
			imageName: "my-invocation-image",
			expected:  "my-invocation-image",
		},
		{
			name:      "specify-image-name-and-namespace",
			imageName: "my-invocation-image",
			expected:  "my-invocation-image",
		},
		{
			name:     "simple-metadata",
			meta:     metadata.AppMetadata{Name: "name", Version: "version"},
			expected: "name:version-invoc",
		},
		{
			name:     "simple-metadata-with-overridden-namespace",
			meta:     metadata.AppMetadata{Name: "name", Version: "version"},
			expected: "name:version-invoc",
		},
		{
			name:     "metadata-with-namespace",
			meta:     metadata.AppMetadata{Name: "name", Version: "version"},
			expected: "name:version-invoc",
		},
		{
			name:     "metadata-with-namespace-and-overridden-namespace",
			meta:     metadata.AppMetadata{Name: "name", Version: "version"},
			expected: "name:version-invoc",
		},
		{
			name: "simple-metadata",
			meta: metadata.AppMetadata{Name: "WrongName&%*", Version: "version"},
			err:  "invalid",
		},
	}
	for _, c := range testcases {
		t.Run(c.name, func(t *testing.T) {
			actual, err := makeImageName(c.meta, c.imageName, "-invoc")
			if c.err != "" {
				assert.ErrorContains(t, err, c.err)
				assert.Equal(t, actual, "")
			} else {
				assert.NilError(t, err)
				assert.Equal(t, actual, c.expected)
			}
		})
	}
}
