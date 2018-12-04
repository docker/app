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
		namespace string
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
			namespace: "my-namespace",
			expected:  "my-namespace/my-invocation-image",
		},
		{
			name:     "simple-metadata",
			meta:     metadata.AppMetadata{Name: "name", Version: "version"},
			expected: "name:version-invoc",
		},
		{
			name:      "simple-metadata-with-overridden-namespace",
			namespace: "my-namespace",
			meta:      metadata.AppMetadata{Name: "name", Version: "version"},
			expected:  "my-namespace/name:version-invoc",
		},
		{
			name:     "metadata-with-namespace",
			meta:     metadata.AppMetadata{Name: "name", Version: "version", Namespace: "namespace"},
			expected: "namespace/name:version-invoc",
		},
		{
			name:      "metadata-with-namespace-and-overridden-namespace",
			namespace: "my-namespace",
			meta:      metadata.AppMetadata{Name: "name", Version: "version", Namespace: "namespace"},
			expected:  "my-namespace/name:version-invoc",
		},
		{
			name: "simple-metadata",
			meta: metadata.AppMetadata{Name: "WrongName&%*", Version: "version"},
			err:  "invalid",
		},
	}
	for _, c := range testcases {
		t.Run(c.name, func(t *testing.T) {
			actual, err := makeInvocationImageName(c.meta, c.namespace, c.imageName)
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

func TestCheckAppImage(t *testing.T) {
	testcases := []struct {
		name      string
		meta      metadata.AppMetadata
		namespace string
		err       string
	}{
		{
			name: "metadata-with-namespace",
			meta: metadata.AppMetadata{Name: "name", Version: "version", Namespace: "namespace"},
		},
		{
			name:      "metadata-with-wrong-namespace",
			meta:      metadata.AppMetadata{Name: "name", Version: "version", Namespace: "namespace"},
			namespace: "WrongNamespace&%*",
			err:       "invalid",
		},
		{
			name: "simple-metadata",
			meta: metadata.AppMetadata{Name: "WrongName&%*", Version: "version"},
			err:  "invalid",
		},
	}
	for _, c := range testcases {
		t.Run(c.name, func(t *testing.T) {
			err := checkAppImage(c.meta, c.namespace)
			if c.err != "" {
				assert.ErrorContains(t, err, c.err)
			} else {
				assert.NilError(t, err)
			}
		})
	}
}
