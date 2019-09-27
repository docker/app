package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/deislabs/cnab-go/bundle"
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

func TestPathToReference(t *testing.T) {
	bundleStore := &bundleStore{path: "base-dir"}

	for _, tc := range []struct {
		Name          string
		Path          string
		ExpectedError string
		ExpectedName  string
	}{
		{
			Name:          "error on invalid path",
			Path:          "invalid",
			ExpectedError: `invalid path "invalid" in the bundle store`,
		}, {
			Name:          "error if file is not json",
			Path:          "registry/repo/name/_tags/file.xml",
			ExpectedError: `invalid path "registry/repo/name/_tags/file.xml", not referencing a CNAB bundle in json format`,
		}, {
			Name:         "return a reference from tagged",
			Path:         "docker.io/library/foo/_tags/latest.json",
			ExpectedName: "docker.io/library/foo",
		}, {
			Name:         "return a reference from digested",
			Path:         "docker.io/library/foo/_digests/sha256/" + testSha + ".json",
			ExpectedName: "docker.io/library/foo",
		},
	} {
		t.Run(tc.Name, func(t *testing.T) {
			ref, err := bundleStore.pathToReference(tc.Path)

			if tc.ExpectedError != "" {
				assert.Equal(t, err.Error(), tc.ExpectedError)
			} else {
				assert.NilError(t, err)
			}

			if tc.ExpectedName != "" {
				assert.Equal(t, ref.Name(), tc.ExpectedName)
			}
		})
	}
}

func TestList(t *testing.T) {
	dockerConfigDir := fs.NewDir(t, t.Name(), fs.WithMode(0755))
	defer dockerConfigDir.Remove()
	appstore, err := NewApplicationStore(dockerConfigDir.Path())
	assert.NilError(t, err)
	bundleStore, err := appstore.BundleStore()
	assert.NilError(t, err)

	refs := []reference.Named{
		parseRefOrDie(t, "my-repo/a-bundle:my-tag"),
		parseRefOrDie(t, "my-repo/b-bundle@sha256:"+testSha),
	}

	t.Run("returns 0 bundles on empty store", func(t *testing.T) {
		bundles, err := bundleStore.List()
		assert.NilError(t, err)
		assert.Equal(t, len(bundles), 0)
	})

	bndl := &bundle.Bundle{Name: "bundle-name"}
	for _, ref := range refs {
		err = bundleStore.Store(ref, bndl)
		assert.NilError(t, err)
	}

	t.Run("Returns the bundles sorted by name", func(t *testing.T) {
		bundles, err := bundleStore.List()
		assert.NilError(t, err)
		assert.Equal(t, len(bundles), 2)
		assert.Equal(t, bundles[0].String(), "docker.io/my-repo/a-bundle:my-tag")
		assert.Equal(t, bundles[1].String(), "docker.io/my-repo/b-bundle@sha256:"+testSha)
	})
}

func TestRemove(t *testing.T) {
	dockerConfigDir := fs.NewDir(t, t.Name(), fs.WithMode(0755))
	defer dockerConfigDir.Remove()
	appstore, err := NewApplicationStore(dockerConfigDir.Path())
	assert.NilError(t, err)
	bundleStore, err := appstore.BundleStore()
	assert.NilError(t, err)

	refs := []reference.Named{
		parseRefOrDie(t, "my-repo/a-bundle:my-tag"),
		parseRefOrDie(t, "my-repo/b-bundle@sha256:"+testSha),
	}

	bndl := &bundle.Bundle{Name: "bundle-name"}
	for _, ref := range refs {
		err = bundleStore.Store(ref, bndl)
		assert.NilError(t, err)
	}

	t.Run("error on unknown", func(t *testing.T) {
		err := bundleStore.Remove(parseRefOrDie(t, "my-repo/some-bundle:1.0.0"))
		assert.Equal(t, err.Error(), "no such image docker.io/my-repo/some-bundle:1.0.0")
	})

	t.Run("remove tagged and digested", func(t *testing.T) {
		bundles, err := bundleStore.List()
		assert.NilError(t, err)
		assert.Equal(t, len(bundles), 2)

		err = bundleStore.Remove(refs[0])

		// Once removed there should be none left
		assert.NilError(t, err)
		bundles, err = bundleStore.List()
		assert.NilError(t, err)
		assert.Equal(t, len(bundles), 1)

		err = bundleStore.Remove(refs[1])
		assert.NilError(t, err)

		bundles, err = bundleStore.List()
		assert.NilError(t, err)
		assert.Equal(t, len(bundles), 0)
	})
}
