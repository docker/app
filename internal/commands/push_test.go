package commands

import (
	"testing"

	"github.com/deislabs/cnab-go/bundle"
	"github.com/docker/app/types/metadata"
	"github.com/docker/distribution/reference"
	"gotest.tools/assert"
)

type retagTestCase struct {
	name                string
	metaName            string
	metaVersion         string
	invocationImageName string
	tag                 string
	bundleRef           string
	expectedImageRef    reference.Named
	expectedCnabRef     reference.Named
	shouldRetag         bool
	errorMessage        string
}

func (c *retagTestCase) buildPackageMetaAndBundle() (metadata.AppMetadata, *bundle.Bundle) {
	return metadata.AppMetadata{Name: c.metaName, Version: c.metaVersion},
		&bundle.Bundle{
			InvocationImages: []bundle.InvocationImage{
				{BaseImage: bundle.BaseImage{Image: c.invocationImageName}},
			},
		}
}

func parseRefOrDie(t *testing.T, name string) reference.Named {
	t.Helper()
	ref, err := reference.ParseNormalizedNamed(name)
	assert.NilError(t, err)
	return ref
}

func TestInvocationImageRetag(t *testing.T) {
	cases := []retagTestCase{
		{
			name:                "no-tag-override-should-not-retag",
			metaName:            "app",
			metaVersion:         "0.1.0",
			invocationImageName: "app:0.1.0-invoc",
			tag:                 "",
			expectedImageRef:    parseRefOrDie(t, "app:0.1.0-invoc"),
			expectedCnabRef:     parseRefOrDie(t, "app:0.1.0"),
			shouldRetag:         false,
		},
		{
			name:                "name-override-should-retag",
			metaName:            "app",
			metaVersion:         "0.1.0",
			invocationImageName: "app:0.1.0-invoc",
			tag:                 "some-app",
			expectedImageRef:    parseRefOrDie(t, "some-app:latest-invoc"),
			expectedCnabRef:     parseRefOrDie(t, "some-app:latest"),
			shouldRetag:         true,
		},
		{
			name:                "tag-override-should-retag",
			metaName:            "app",
			metaVersion:         "0.1.0",
			invocationImageName: "app:0.1.0-invoc",
			tag:                 "some-app:test",
			expectedImageRef:    parseRefOrDie(t, "some-app:test-invoc"),
			expectedCnabRef:     parseRefOrDie(t, "some-app:test"),
			shouldRetag:         true,
		},
		{
			name:                "bundle-ref-should-retag",
			metaName:            "app",
			metaVersion:         "0.1.0",
			invocationImageName: "app:0.1.0-invoc",
			bundleRef:           "some-app:test",
			expectedImageRef:    parseRefOrDie(t, "some-app:test-invoc"),
			expectedCnabRef:     parseRefOrDie(t, "some-app:test"),
			shouldRetag:         true,
		},
		{
			name:                "tag-overrides-bundle-ref",
			metaName:            "app",
			metaVersion:         "0.1.0",
			invocationImageName: "app:0.1.0-invoc",
			tag:                 "some-app:test",
			bundleRef:           "other-app:other-test",
			expectedImageRef:    parseRefOrDie(t, "some-app:test-invoc"),
			expectedCnabRef:     parseRefOrDie(t, "some-app:test"),
			shouldRetag:         true,
		},
		{
			name:                "parsing-error",
			metaName:            "app",
			metaVersion:         "0.1.0",
			invocationImageName: "app:0.1.0-invoc",
			tag:                 "some-App:test",
			errorMessage:        "some-App:test: invalid reference format: repository name must be lowercase",
		},
		{
			name:                "error-no-digest",
			metaName:            "app",
			metaVersion:         "0.1.0",
			invocationImageName: "app:0.1.0-invoc",
			tag:                 "some-app@sha256:424d908f6801f786c68341b5d9083858f784142eb7a521a1512e0407e3ac8e75",
			errorMessage:        "some-app@sha256:424d908f6801f786c68341b5d9083858f784142eb7a521a1512e0407e3ac8e75: can't push to a digested reference",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p, b := c.buildPackageMetaAndBundle()
			retag, err := shouldRetagInvocationImage(p, b, c.tag, c.bundleRef)
			if c.errorMessage == "" {
				assert.NilError(t, err)
				assert.Equal(t, retag.shouldRetag, c.shouldRetag)
				assert.Equal(t, retag.invocationImageRef.String(), c.expectedImageRef.String())
				assert.Equal(t, retag.cnabRef.String(), c.expectedCnabRef.String())
			} else {
				assert.ErrorContains(t, err, c.errorMessage)
			}
		})
	}
}

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
