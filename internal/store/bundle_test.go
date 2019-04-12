package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/deislabs/duffle/pkg/bundle"
	"github.com/docker/distribution/reference"
	"gotest.tools/assert"
	"gotest.tools/fs"
)

const (
	testSha = "2957c6606cc94099f7dfe0011b5c8daf4a605ed6124d4eee773bab1e05a8ce87"
)

func TestStoreAndReadBundle(t *testing.T) {
	dockerConfigDir := fs.NewDir(t, t.Name(), fs.WithMode(0755))
	defer dockerConfigDir.Remove()
	appstore, err := NewApplicationStore(dockerConfigDir.Path())
	assert.NilError(t, err)
	bundleStore, err := appstore.BundleStore()
	assert.NilError(t, err)

	expectedBundle := &bundle.Bundle{Name: "bundle-name"}

	testcases := []struct {
		name string
		ref  reference.Named
		path string
	}{
		{
			name: "tagged",
			ref:  parseRefOrDie(t, "my-repo/my-bundle:my-tag"),
			path: dockerConfigDir.Join("app", "bundles", "docker.io", "my-repo", "my-bundle", "_tags", "my-tag.json"),
		},
		{
			name: "digested",
			ref:  parseRefOrDie(t, "my-repo/my-bundle@sha256:"+testSha),
			path: dockerConfigDir.Join("app", "bundles", "docker.io", "my-repo", "my-bundle", "_digests", "sha256", testSha+".json"),
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {
			// Store the bundle
			err = bundleStore.Store(testcase.ref, expectedBundle)
			assert.NilError(t, err)

			// Check the file exists
			_, err = os.Stat(testcase.path)
			assert.NilError(t, err)

			// Load it
			actualBundle, err := bundleStore.Read(testcase.ref)
			assert.NilError(t, err)
			assert.DeepEqual(t, expectedBundle, actualBundle)
		})
	}
}

func parseRefOrDie(t *testing.T, ref string) reference.Named {
	t.Helper()
	named, err := reference.ParseNormalizedNamed(ref)
	assert.NilError(t, err)
	return named
}

func TestStorePath(t *testing.T) {
	bs := &bundleStore{path: "base-dir"}
	for _, tc := range []struct {
		Name            string
		Ref             reference.Named
		ExpectedSubpath string
		ExpectedError   string
	}{
		// storePath expects a tagged or digested, i.e. the use of TagNameOnly to add :latest. Check that it rejects untagged refs
		{
			Name:          "untagged",
			Ref:           parseRefOrDie(t, "foo"),
			ExpectedError: "docker.io/library/foo: not tagged or digested",
		},
		// Variants of a tagged ref
		{
			Name:            "simple-tagged",
			Ref:             parseRefOrDie(t, "foo:latest"),
			ExpectedSubpath: "docker.io/library/foo/_tags/latest.json",
		},
		{
			Name:            "deep-simple-tagged",
			Ref:             parseRefOrDie(t, "user/foo/bar:latest"),
			ExpectedSubpath: "docker.io/user/foo/bar/_tags/latest.json",
		},
		{
			Name:            "host-and-tagged",
			Ref:             parseRefOrDie(t, "my.registry.example.com/foo:latest"),
			ExpectedSubpath: "my.registry.example.com/foo/_tags/latest.json",
		},
		{
			Name:            "host-port-and-tagged",
			Ref:             parseRefOrDie(t, "my.registry.example.com:5000/foo:latest"),
			ExpectedSubpath: "my.registry.example.com_5000/foo/_tags/latest.json",
		},
		// Variants of a digested ref
		{
			Name:            "simple-digested",
			Ref:             parseRefOrDie(t, "foo@sha256:"+testSha),
			ExpectedSubpath: "docker.io/library/foo/_digests/sha256/" + testSha + ".json",
		},
		{
			Name:            "deep-simple-digested",
			Ref:             parseRefOrDie(t, "user/foo/bar@sha256:"+testSha),
			ExpectedSubpath: "docker.io/user/foo/bar/_digests/sha256/" + testSha + ".json",
		},
		{
			Name:            "host-and-digested",
			Ref:             parseRefOrDie(t, "my.registry.example.com/foo@sha256:"+testSha),
			ExpectedSubpath: "my.registry.example.com/foo/_digests/sha256/" + testSha + ".json",
		},
		{
			Name:            "host-port-and-digested",
			Ref:             parseRefOrDie(t, "my.registry.example.com:5000/foo@sha256:"+testSha),
			ExpectedSubpath: "my.registry.example.com_5000/foo/_digests/sha256/" + testSha + ".json",
		},
		// If both then digest takes precedence (tag is ignored)
		{
			Name:            "simple-tagged-and-digested",
			Ref:             parseRefOrDie(t, "foo:latest@sha256:"+testSha),
			ExpectedSubpath: "docker.io/library/foo/_digests/sha256/" + testSha + ".json",
		},
		{
			Name:            "deep-simple-tagged-and-digested",
			Ref:             parseRefOrDie(t, "user/foo/bar:latest@sha256:"+testSha),
			ExpectedSubpath: "docker.io/user/foo/bar/_digests/sha256/" + testSha + ".json",
		},
		{
			Name:            "host-and-tagged-and-digested",
			Ref:             parseRefOrDie(t, "my.registry.example.com/foo:latest@sha256:"+testSha),
			ExpectedSubpath: "my.registry.example.com/foo/_digests/sha256/" + testSha + ".json",
		},
		{
			Name:            "host-port-and-tagged-and-digested",
			Ref:             parseRefOrDie(t, "my.registry.example.com:5000/foo:latest@sha256:"+testSha),
			ExpectedSubpath: "my.registry.example.com_5000/foo/_digests/sha256/" + testSha + ".json",
		},
	} {
		t.Run(tc.Name, func(t *testing.T) {
			path, err := bs.storePath(tc.Ref)
			if tc.ExpectedError == "" {
				assert.NilError(t, err)
				assert.Equal(t, filepath.Join("base-dir", filepath.FromSlash(tc.ExpectedSubpath)), path)
			} else {
				assert.Error(t, err, tc.ExpectedError)
			}
		})
	}
}
