package store

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/docker/app/internal/relocated"

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

	expectedBundle := relocated.FromBundle(&bundle.Bundle{Name: "bundle-name"})

	testcases := []struct {
		name string
		ref  reference.Named
		path string
	}{
		{
			name: "tagged",
			ref:  parseRefOrDie(t, "my-repo/my-bundle:my-tag"),
			path: dockerConfigDir.Join("app", "bundles", "docker.io", "my-repo", "my-bundle", "_tags", "my-tag", relocated.BundleFilename),
		},
		{
			name: "digested",
			ref:  parseRefOrDie(t, "my-repo/my-bundle@sha256:"+testSha),
			path: dockerConfigDir.Join("app", "bundles", "docker.io", "my-repo", "my-bundle", "_digests", "sha256", testSha, relocated.BundleFilename),
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {
			// Store the bundle
			_, err = bundleStore.Store(expectedBundle, testcase.ref)
			assert.NilError(t, err)

			// Check the file exists
			_, err = os.Stat(testcase.path)
			assert.NilError(t, err)

			// Load it
			actualBundle, err := bundleStore.Read(testcase.ref)
			assert.NilError(t, err)
			assert.DeepEqual(t, expectedBundle.Bundle, actualBundle.Bundle)
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
			ExpectedSubpath: "docker.io/library/foo/_tags/latest",
		},
		{
			Name:            "deep-simple-tagged",
			Ref:             parseRefOrDie(t, "user/foo/bar:latest"),
			ExpectedSubpath: "docker.io/user/foo/bar/_tags/latest",
		},
		{
			Name:            "host-and-tagged",
			Ref:             parseRefOrDie(t, "my.registry.example.com/foo:latest"),
			ExpectedSubpath: "my.registry.example.com/foo/_tags/latest",
		},
		{
			Name:            "host-port-and-tagged",
			Ref:             parseRefOrDie(t, "my.registry.example.com:5000/foo:latest"),
			ExpectedSubpath: "my.registry.example.com_5000/foo/_tags/latest",
		},
		// Variants of a digested ref
		{
			Name:            "simple-digested",
			Ref:             parseRefOrDie(t, "foo@sha256:"+testSha),
			ExpectedSubpath: "docker.io/library/foo/_digests/sha256/" + testSha,
		},
		{
			Name:            "deep-simple-digested",
			Ref:             parseRefOrDie(t, "user/foo/bar@sha256:"+testSha),
			ExpectedSubpath: "docker.io/user/foo/bar/_digests/sha256/" + testSha,
		},
		{
			Name:            "host-and-digested",
			Ref:             parseRefOrDie(t, "my.registry.example.com/foo@sha256:"+testSha),
			ExpectedSubpath: "my.registry.example.com/foo/_digests/sha256/" + testSha,
		},
		{
			Name:            "host-port-and-digested",
			Ref:             parseRefOrDie(t, "my.registry.example.com:5000/foo@sha256:"+testSha),
			ExpectedSubpath: "my.registry.example.com_5000/foo/_digests/sha256/" + testSha,
		},
		// If both then digest takes precedence (tag is ignored)
		{
			Name:            "simple-tagged-and-digested",
			Ref:             parseRefOrDie(t, "foo:latest@sha256:"+testSha),
			ExpectedSubpath: "docker.io/library/foo/_digests/sha256/" + testSha,
		},
		{
			Name:            "deep-simple-tagged-and-digested",
			Ref:             parseRefOrDie(t, "user/foo/bar:latest@sha256:"+testSha),
			ExpectedSubpath: "docker.io/user/foo/bar/_digests/sha256/" + testSha,
		},
		{
			Name:            "host-and-tagged-and-digested",
			Ref:             parseRefOrDie(t, "my.registry.example.com/foo:latest@sha256:"+testSha),
			ExpectedSubpath: "my.registry.example.com/foo/_digests/sha256/" + testSha,
		},
		{
			Name:            "host-port-and-tagged-and-digested",
			Ref:             parseRefOrDie(t, "my.registry.example.com:5000/foo:latest@sha256:"+testSha),
			ExpectedSubpath: "my.registry.example.com_5000/foo/_digests/sha256/" + testSha,
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
			Path:         "docker.io/library/foo/_tags/latest/bundle.json",
			ExpectedName: "docker.io/library/foo",
		}, {
			Name:         "return a reference from digested",
			Path:         "docker.io/library/foo/_digests/sha256/" + testSha + "/bundle.json",
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

	bndl := relocated.FromBundle(&bundle.Bundle{Name: "bundle-name"})
	for _, ref := range refs {
		_, err = bundleStore.Store(bndl, ref)
		assert.NilError(t, err)
	}

	t.Run("Returns the bundles sorted by name", func(t *testing.T) {
		bundles, err := bundleStore.List()
		assert.NilError(t, err)
		assert.Equal(t, len(bundles), 2)
		assert.Equal(t, bundles[0].String(), "docker.io/my-repo/a-bundle:my-tag")
		assert.Equal(t, bundles[1].String(), "docker.io/my-repo/b-bundle@sha256:"+testSha)
	})

	t.Run("Ignores unknown files in the bundle store", func(t *testing.T) {
		p := path.Join(dockerConfigDir.Path(), AppConfigDirectory, BundleStoreDirectory)
		//nolint:errcheck
		os.OpenFile(path.Join(p, "filename"), os.O_CREATE, 06444)

		bundles, err := bundleStore.List()
		assert.NilError(t, err)
		assert.Equal(t, len(bundles), 2)
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

	bndl := relocated.FromBundle(&bundle.Bundle{Name: "bundle-name"})
	for _, ref := range refs {
		_, err = bundleStore.Store(bndl, ref)
		assert.NilError(t, err)
	}

	t.Run("error on unknown", func(t *testing.T) {
		err := bundleStore.Remove(parseRefOrDie(t, "my-repo/some-bundle:1.0.0"), false)
		assert.Equal(t, err.Error(), "no such image my-repo/some-bundle:1.0.0")
	})

	t.Run("remove tagged and digested", func(t *testing.T) {
		bundles, err := bundleStore.List()
		assert.NilError(t, err)
		assert.Equal(t, len(bundles), 2)

		err = bundleStore.Remove(refs[0], false)

		// Once removed there should be none left
		assert.NilError(t, err)
		bundles, err = bundleStore.List()
		assert.NilError(t, err)
		assert.Equal(t, len(bundles), 1)

		err = bundleStore.Remove(refs[1], false)
		assert.NilError(t, err)

		bundles, err = bundleStore.List()
		assert.NilError(t, err)
		assert.Equal(t, len(bundles), 0)
	})
}

func TestRemoveById(t *testing.T) {
	dockerConfigDir := fs.NewDir(t, t.Name(), fs.WithMode(0755))
	defer dockerConfigDir.Remove()
	appstore, err := NewApplicationStore(dockerConfigDir.Path())
	assert.NilError(t, err)
	bundleStore, err := appstore.BundleStore()
	assert.NilError(t, err)

	t.Run("error when id does not exist", func(t *testing.T) {
		idRef, err := FromBundle(relocated.FromBundle(&bundle.Bundle{Name: "not-stored-bundle-name"}))
		assert.NilError(t, err)

		err = bundleStore.Remove(idRef, false)
		assert.Equal(t, err.Error(), fmt.Sprintf("no such image %q", reference.FamiliarString(idRef)))
	})

	t.Run("error on multiple repositories", func(t *testing.T) {
		bndl := relocated.FromBundle(&bundle.Bundle{Name: "bundle-name"})
		idRef, err := FromBundle(bndl)
		assert.NilError(t, err)
		_, err = bundleStore.Store(bndl, parseRefOrDie(t, "my-repo/a-bundle:my-tag"))
		assert.NilError(t, err)
		_, err = bundleStore.Store(bndl, parseRefOrDie(t, "my-repo/b-bundle:my-tag"))
		assert.NilError(t, err)

		err = bundleStore.Remove(idRef, false)
		assert.Equal(t, err.Error(), fmt.Sprintf("unable to delete %q - App is referenced in multiple repositories", reference.FamiliarString(idRef)))
	})

	t.Run("success on multiple repositories but force", func(t *testing.T) {
		bndl := relocated.FromBundle(&bundle.Bundle{Name: "bundle-name"})
		idRef, err := FromBundle(bndl)
		assert.NilError(t, err)
		_, err = bundleStore.Store(bndl, parseRefOrDie(t, "my-repo/a-bundle:my-tag"))
		assert.NilError(t, err)
		_, err = bundleStore.Store(bndl, parseRefOrDie(t, "my-repo/b-bundle:my-tag"))
		assert.NilError(t, err)

		err = bundleStore.Remove(idRef, true)
		assert.NilError(t, err)
	})

	t.Run("success when only one reference exists", func(t *testing.T) {
		bndl := relocated.FromBundle(&bundle.Bundle{Name: "other-bundle-name"})
		ref := parseRefOrDie(t, "my-repo/other-bundle:my-tag")
		_, err = bundleStore.Store(bndl, ref)

		idRef, err := FromBundle(bndl)
		assert.NilError(t, err)

		err = bundleStore.Remove(idRef, false)
		assert.NilError(t, err)
		bundles, err := bundleStore.List()
		assert.NilError(t, err)
		for _, bref := range bundles {
			assert.Equal(t, bref == ref, false)
		}
	})
}
func TestLookUp(t *testing.T) {
	dockerConfigDir := fs.NewDir(t, t.Name(), fs.WithMode(0755))
	defer dockerConfigDir.Remove()
	appstore, err := NewApplicationStore(dockerConfigDir.Path())
	assert.NilError(t, err)
	bundleStore, err := appstore.BundleStore()
	assert.NilError(t, err)
	bndl := relocated.FromBundle(&bundle.Bundle{Name: "bundle-name"})
	// Adding the bundle referenced by id
	id, err := bundleStore.Store(bndl, nil)
	assert.NilError(t, err)
	// Adding the same bundle referenced by a tag
	ref := parseRefOrDie(t, "my-repo/a-bundle:my-tag")
	_, err = bundleStore.Store(bndl, ref)
	assert.NilError(t, err)
	// Adding the same bundle referenced by tag prefixed by docker.io/library
	dockerIoRef := parseRefOrDie(t, "docker.io/library/a-bundle:my-tag")
	_, err = bundleStore.Store(bndl, dockerIoRef)
	assert.NilError(t, err)

	for _, tc := range []struct {
		Name          string
		refOrID       string
		ExpectedError string
		ExpectedRef   reference.Reference
	}{
		{
			Name:          "Long Id",
			refOrID:       id.String(),
			ExpectedError: "",
			ExpectedRef:   id,
		},
		{
			Name:          "Short Id",
			refOrID:       id.String()[0:8],
			ExpectedError: "",
			ExpectedRef:   id,
		},
		{
			Name:          "Tagged Ref",
			refOrID:       "my-repo/a-bundle:my-tag",
			ExpectedError: "",
			ExpectedRef:   ref,
		},
		{
			Name:          "docker.io_library repository Tagged Ref",
			refOrID:       "a-bundle:my-tag",
			ExpectedError: "",
			ExpectedRef:   dockerIoRef,
		},
		{
			Name:          "Unknown Tag",
			refOrID:       "other-repo/a-bundle:other-tag",
			ExpectedError: "other-repo/a-bundle:other-tag: reference not found",
			ExpectedRef:   nil,
		},
		{
			Name:          "Unknown ID",
			refOrID:       "b4fcc3af16804e29d977918a3a322daf1eb6ab2992c3cc7cbfeae8c3d6ede8af",
			ExpectedError: "b4fcc3af16804e29d977918a3a322daf1eb6ab2992c3cc7cbfeae8c3d6ede8af: reference not found",
			ExpectedRef:   nil,
		},
		{
			Name:          "Unknown short ID",
			refOrID:       "b4fcc3af",
			ExpectedError: "b4fcc3af:latest: reference not found",
			ExpectedRef:   nil,
		},
	} {
		t.Run(tc.Name, func(t *testing.T) {
			fmt.Println(tc.refOrID)
			ref, err := bundleStore.LookUp(tc.refOrID)

			if tc.ExpectedError != "" {
				assert.Equal(t, err.Error(), tc.ExpectedError)
			} else {
				assert.NilError(t, err)
			}

			if tc.ExpectedRef != nil {
				assert.Equal(t, ref, tc.ExpectedRef)
			}
		})
	}
}

func TestScanBundles(t *testing.T) {
	dockerConfigDir := fs.NewDir(t, t.Name(), fs.WithMode(0755))
	defer dockerConfigDir.Remove()

	// Adding a bundle which should be referenced by id only
	bndl1 := relocated.FromBundle(&bundle.Bundle{Name: "bundle-1"})
	id1, err := FromBundle(bndl1)
	assert.NilError(t, err)
	dir1 := dockerConfigDir.Join("app", "bundles", "_ids", id1.String())
	assert.NilError(t, os.MkdirAll(dir1, 0755))
	assert.NilError(t, ioutil.WriteFile(filepath.Join(dir1, relocated.BundleFilename), []byte(`{"name": "bundle-1"}`), 0644))

	// Adding a bundle which should be referenced by id and tag
	bndl2 := relocated.FromBundle(&bundle.Bundle{Name: "bundle-2"})
	id2, err := FromBundle(bndl2)
	assert.NilError(t, err)
	dir2 := dockerConfigDir.Join("app", "bundles", "_ids", id2.String())
	assert.NilError(t, os.MkdirAll(dir2, 0755))
	assert.NilError(t, ioutil.WriteFile(filepath.Join(dir2, relocated.BundleFilename), []byte(`{"name": "bundle-2"}`), 0644))
	dir2 = dockerConfigDir.Join("app", "bundles", "docker.io", "my-repo", "my-bundle", "_tags", "my-tag")
	assert.NilError(t, os.MkdirAll(dir2, 0755))
	assert.NilError(t, ioutil.WriteFile(filepath.Join(dir2, relocated.BundleFilename), []byte(`{"name": "bundle-2"}`), 0644))

	appstore, err := NewApplicationStore(dockerConfigDir.Path())
	assert.NilError(t, err)
	bundleStore, err := appstore.BundleStore()
	assert.NilError(t, err)

	// Ensure List() and Read() function returns expected bundles
	refs, err := bundleStore.List()
	assert.NilError(t, err)
	expectedRefs := []string{id2.String(), "my-repo/my-bundle:my-tag", id1.String()}
	refsAsString := func(references []reference.Reference) []string {
		var rv []string
		for _, r := range references {
			rv = append(rv, reference.FamiliarString(r))
		}
		return rv
	}
	assert.DeepEqual(t, refsAsString(refs), expectedRefs)
	bndl, err := bundleStore.Read(id1)
	assert.NilError(t, err)
	assert.DeepEqual(t, bndl, bndl1)
	bndl, err = bundleStore.Read(id2)
	assert.NilError(t, err)
	assert.DeepEqual(t, bndl, bndl2)
	bndl, err = bundleStore.Read(parseRefOrDie(t, "my-repo/my-bundle:my-tag"))
	assert.NilError(t, err)
	assert.DeepEqual(t, bndl, bndl2)
}

func TestAppendRemoveReference(t *testing.T) {
	id1, err := FromString("68720b2db729794a3521bc83e3699ac629f26beba6862b6ec491cd0d677d02a0")
	assert.NilError(t, err)
	id2, err := FromString("b7244e15970354cceb75f417f1e98b3a340cff35576eeeac603d33afa73b0b4b")
	assert.NilError(t, err)
	initialize := func() referencesMap {
		m := make(referencesMap)
		m[id1] = []reference.Reference{id1, parseRefOrDie(t, "foo/bar:1.0")}
		m[id2] = []reference.Reference{parseRefOrDie(t, "qix/qux:1.0")}
		return m
	}

	t.Run("Add new reference to existing ID", func(t *testing.T) {
		m := initialize()
		ref := parseRefOrDie(t, "zox/zox:1.0")
		m.appendRef(id2, ref)
		assert.Equal(t, len(m), 2)
		assert.Equal(t, len(m[id2]), 2)
		assert.Equal(t, m[id2][1], ref)
	})

	t.Run("Add new ID", func(t *testing.T) {
		m := initialize()
		id3, err := FromString("d19b0b0d9ac36a8198465bcd8bf816a45110bf26b731a4e299e771ca0b082a21")
		assert.NilError(t, err)
		ref := parseRefOrDie(t, "zox/zox:1.0")
		m.appendRef(id3, ref)
		assert.Equal(t, len(m), 3)
		assert.Equal(t, len(m[id3]), 1)
		assert.Equal(t, m[id3][0], ref)
	})

	t.Run("Remove reference", func(t *testing.T) {
		m := initialize()
		ref := parseRefOrDie(t, "foo/bar:1.0")
		fmt.Println(m)
		m.removeRef(ref)
		assert.Equal(t, len(m), 2)
		assert.Equal(t, len(m[id1]), 1)
		assert.Equal(t, m[id1][0], id1)
	})

	t.Run("Remove reference and ID", func(t *testing.T) {
		m := initialize()
		ref := parseRefOrDie(t, "qix/qux:1.0")
		fmt.Println(m)
		m.removeRef(ref)
		assert.Equal(t, len(m), 1)
		_, exist := m[id2]
		assert.Equal(t, exist, false)
	})
}
