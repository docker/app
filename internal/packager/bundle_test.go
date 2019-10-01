package packager

import (
	"testing"

	"github.com/docker/app/types/metadata"
	"gotest.tools/assert"
)

func TestMakeInvocationImage(t *testing.T) {
	testcases := []struct {
		name     string
		meta     metadata.AppMetadata
		tag      string
		expected string
		err      string
	}{
		{
			name:     "simple-metadata",
			meta:     metadata.AppMetadata{Name: "name", Version: "version"},
			expected: "name:version-invoc",
		},
		{
			name:     "tag-override",
			meta:     metadata.AppMetadata{Name: "name", Version: "version"},
			expected: "myimage:mytag-invoc",
			tag:      "myimage:mytag",
		},
		{
			name: "invalid-metadata",
			meta: metadata.AppMetadata{Name: "WrongName&%*", Version: "version"},
			err:  "invalid",
		},
	}
	for _, c := range testcases {
		t.Run(c.name, func(t *testing.T) {
			ref, err := GetNamedTagged(c.tag)
			assert.NilError(t, err)
			actual, err := MakeInvocationImageName(c.meta, ref)
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
