package commands

import (
	"testing"

	"github.com/docker/app/types/metadata"
	"gotest.tools/assert"
)

func TestMakeInvocationImage(t *testing.T) {
	testcases := []struct {
		name     string
		meta     metadata.AppMetadata
		expected string
		err      string
	}{
		{
			name:     "simple-metadata",
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
			actual, err := makeImageName(c.meta)
			if c.err != "" {
				assert.ErrorContains(t, err, c.err)
				assert.Equal(t, actual, "", "On "+c.meta.Name)
			} else {
				assert.NilError(t, err)
				assert.Equal(t, actual, c.expected)
			}
		})
	}
}
