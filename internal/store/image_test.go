package store

import (
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/docker/app/internal/image"

	"github.com/cnabio/cnab-go/bundle"
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
	imageStore, err := appstore.ImageStore()
	assert.NilError(t, err)

	expectedBundle := image.FromBundle(&bundle.Bundle{Name: "bundle-name"})

	testcases := []struct {
		name string
		ref  reference.Named
	}{
		{
			name: "tagged",
			ref:  parseRefOrDie(t, "my-repo/my-bundle:my-tag"),
		},
		{
			name: "digested",
			ref:  parseRefOrDie(t, "my-repo/my-bundle@sha256:"+testSha),
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {
			// Store the bundle
			_, err = imageStore.Store(expectedBundle, testcase.ref)
			assert.NilError(t, err)

			// Load it
			actualBundle, err := imageStore.Read(testcase.ref)
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

func TestList(t *testing.T) {
	dockerConfigDir := fs.NewDir(t, t.Name(), fs.WithMode(0755))
	defer dockerConfigDir.Remove()
	appstore, err := NewApplicationStore(dockerConfigDir.Path())
	assert.NilError(t, err)
	imageStore, err := appstore.ImageStore()
	assert.NilError(t, err)

	refs := []reference.Named{
		parseRefOrDie(t, "my-repo/a-bundle:my-tag"),
		parseRefOrDie(t, "my-repo/b-bundle@sha256:"+testSha),
	}

	t.Run("returns 0 bundles on empty store", func(t *testing.T) {
		bundles, err := imageStore.List()
		assert.NilError(t, err)
		assert.Equal(t, len(bundles), 0)
	})

	img := image.FromBundle(&bundle.Bundle{Name: "bundle-name"})
	for _, ref := range refs {
		_, err = imageStore.Store(img, ref)
		assert.NilError(t, err)
	}

	t.Run("Returns the bundles sorted by name", func(t *testing.T) {
		bundles, err := imageStore.List()
		assert.NilError(t, err)
		assert.Equal(t, len(bundles), 2)
		assert.Equal(t, bundles[0].String(), "docker.io/my-repo/a-bundle:my-tag")
		assert.Equal(t, bundles[1].String(), "docker.io/my-repo/b-bundle@sha256:"+testSha)
	})

	t.Run("Ignores unknown files in the bundle store", func(t *testing.T) {
		p := path.Join(dockerConfigDir.Path(), AppConfigDirectory, ImageStoreDirectory)
		//nolint:errcheck
		os.OpenFile(path.Join(p, "filename"), os.O_CREATE, 06444)

		bundles, err := imageStore.List()
		assert.NilError(t, err)
		assert.Equal(t, len(bundles), 2)
	})
}

func TestRemove(t *testing.T) {
	dockerConfigDir := fs.NewDir(t, t.Name(), fs.WithMode(0755))
	defer dockerConfigDir.Remove()
	appstore, err := NewApplicationStore(dockerConfigDir.Path())
	assert.NilError(t, err)
	imageStore, err := appstore.ImageStore()
	assert.NilError(t, err)

	refs := []reference.Named{
		parseRefOrDie(t, "my-repo/a-bundle:my-tag"),
		parseRefOrDie(t, "my-repo/b-bundle@sha256:"+testSha),
	}

	img := image.FromBundle(&bundle.Bundle{Name: "bundle-name"})
	for _, ref := range refs {
		_, err = imageStore.Store(img, ref)
		assert.NilError(t, err)
	}

	t.Run("error on unknown", func(t *testing.T) {
		err := imageStore.Remove(parseRefOrDie(t, "my-repo/some-bundle:1.0.0"), false)
		assert.Equal(t, err.Error(), "reference does not exist")
	})

	t.Run("remove tagged and digested", func(t *testing.T) {
		bundles, err := imageStore.List()
		assert.NilError(t, err)
		assert.Equal(t, len(bundles), 2)

		err = imageStore.Remove(refs[0], false)

		// Once removed there should be none left
		assert.NilError(t, err)
		bundles, err = imageStore.List()
		assert.NilError(t, err)
		assert.Equal(t, len(bundles), 1)

		err = imageStore.Remove(refs[1], false)
		assert.NilError(t, err)

		bundles, err = imageStore.List()
		assert.NilError(t, err)
		assert.Equal(t, len(bundles), 0)
	})
}

func TestRemoveById(t *testing.T) {
	dockerConfigDir := fs.NewDir(t, t.Name(), fs.WithMode(0755))
	defer dockerConfigDir.Remove()
	appstore, err := NewApplicationStore(dockerConfigDir.Path())
	assert.NilError(t, err)
	imageStore, err := appstore.ImageStore()
	assert.NilError(t, err)

	t.Run("error when id does not exist", func(t *testing.T) {
		idRef, err := FromAppImage(image.FromBundle(&bundle.Bundle{Name: "not-stored-bundle-name"}))
		assert.NilError(t, err)

		err = imageStore.Remove(idRef, false)
		assert.Equal(t, err.Error(), fmt.Sprintf("%s: reference not found", idRef.String()))
	})

	t.Run("error on multiple repositories", func(t *testing.T) {
		img := image.FromBundle(&bundle.Bundle{Name: "bundle-name"})
		idRef, err := FromAppImage(img)
		assert.NilError(t, err)
		_, err = imageStore.Store(img, idRef)
		assert.NilError(t, err)
		_, err = imageStore.Store(img, parseRefOrDie(t, "my-repo/a-bundle:my-tag"))
		assert.NilError(t, err)
		_, err = imageStore.Store(img, parseRefOrDie(t, "my-repo/a-bundle:latest"))
		assert.NilError(t, err)

		err = imageStore.Remove(idRef, false)
		assert.Equal(t, err.Error(), fmt.Sprintf("unable to delete %q - App is referenced in multiple repositories", reference.FamiliarString(idRef)))
	})

	t.Run("success on multiple repositories but force", func(t *testing.T) {
		img := image.FromBundle(&bundle.Bundle{Name: "bundle-name"})
		idRef, err := FromAppImage(img)
		assert.NilError(t, err)
		_, err = imageStore.Store(img, idRef)
		assert.NilError(t, err)
		_, err = imageStore.Store(img, parseRefOrDie(t, "my-repo/a-bundle:my-tag"))
		assert.NilError(t, err)

		err = imageStore.Remove(idRef, true)
		assert.NilError(t, err)
	})

	t.Run("success when only one reference exists", func(t *testing.T) {
		img := image.FromBundle(&bundle.Bundle{Name: "other-bundle-name"})
		ref := parseRefOrDie(t, "my-repo/other-bundle:my-tag")
		_, err = imageStore.Store(img, ref)

		idRef, err := FromAppImage(img)
		assert.NilError(t, err)

		err = imageStore.Remove(idRef, false)
		assert.NilError(t, err)
		bundles, err := imageStore.List()
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
	imageStore, err := appstore.ImageStore()
	assert.NilError(t, err)
	img := image.FromBundle(&bundle.Bundle{Name: "bundle-name"})
	// Adding the bundle referenced by id
	id, err := imageStore.Store(img, nil)
	assert.NilError(t, err)
	// Adding the same bundle referenced by a tag
	ref := parseRefOrDie(t, "my-repo/a-bundle:my-tag")
	_, err = imageStore.Store(img, ref)
	assert.NilError(t, err)
	// Adding the same bundle referenced by tag prefixed by docker.io/library
	dockerIoRef := parseRefOrDie(t, "docker.io/library/a-bundle:my-tag")
	_, err = imageStore.Store(img, dockerIoRef)
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
			ExpectedError: "b4fcc3af: reference not found",
			ExpectedRef:   nil,
		},
	} {
		t.Run(tc.Name, func(t *testing.T) {
			fmt.Println(tc.refOrID)
			ref, err := imageStore.LookUp(tc.refOrID)

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
