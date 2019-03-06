package main

import (
	"testing"

	"github.com/deislabs/duffle/pkg/bundle"
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
			name:                "no-tag-aligned-names",
			metaName:            "app",
			metaVersion:         "0.1.0",
			invocationImageName: "app:0.1.0-invoc",
			tag:                 "",
			expectedImageRef:    parseRefOrDie(t, "app:0.1.0-invoc"),
			expectedCnabRef:     parseRefOrDie(t, "app:0.1.0"),
			shouldRetag:         false,
		},
		{
			name:                "tag-aligned-names-untagged-ref",
			metaName:            "app",
			metaVersion:         "0.1.0",
			invocationImageName: "app:0.1.0-invoc",
			tag:                 "some-app",
			expectedImageRef:    parseRefOrDie(t, "some-app:latest-invoc"),
			expectedCnabRef:     parseRefOrDie(t, "some-app:latest"),
			shouldRetag:         true,
		},
		{
			name:                "tag-aligned-names-tagged-ref",
			metaName:            "app",
			metaVersion:         "0.1.0",
			invocationImageName: "app:0.1.0-invoc",
			tag:                 "some-app:test",
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
			name:                "no-digest",
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
			retag, err := shouldRetagInvocationImage(p, b, c.tag)
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
