package image

import (
	"bufio"
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/docker/app/internal/relocated"

	"gotest.tools/assert"

	"github.com/deislabs/cnab-go/bundle"
	"github.com/docker/app/internal/store"
	"github.com/docker/cli/cli/command"
	"github.com/docker/distribution/reference"
)

type bundleStoreStubForListCmd struct {
	refMap map[reference.Reference]*relocated.Bundle
	// in order to keep the reference in the same order between tests
	refList []reference.Reference
}

func (b *bundleStoreStubForListCmd) Store(ref reference.Reference, bndl *relocated.Bundle) (reference.Digested, error) {
	b.refMap[ref] = bndl
	b.refList = append(b.refList, ref)
	return store.FromBundle(bndl)
}

func (b *bundleStoreStubForListCmd) Read(ref reference.Reference) (*relocated.Bundle, error) {
	bndl, ok := b.refMap[ref]
	if ok {
		return bndl, nil
	}
	return nil, fmt.Errorf("Bundle not found")
}

func (b *bundleStoreStubForListCmd) List() ([]reference.Reference, error) {
	return b.refList, nil
}

func (b *bundleStoreStubForListCmd) Remove(ref reference.Reference) error {
	return nil
}

func (b *bundleStoreStubForListCmd) LookUp(refOrID string) (reference.Reference, error) {
	return nil, nil
}

func TestListCmd(t *testing.T) {
	ref, err := store.FromString("a855ac937f2ed375ba4396bbc49c4093e124da933acd2713fb9bc17d7562a087")
	assert.NilError(t, err)
	refs := []reference.Reference{
		parseReference(t, "foo/bar@sha256:b59492bb814012ca3d2ce0b6728242d96b4af41687cc82166a4b5d7f2d9fb865"),
		parseReference(t, "foo/bar:1.0"),
		ref,
	}
	bundles := []relocated.Bundle{
		{
			Bundle: &bundle.Bundle{
				Name: "Digested App",
			},
		},
		{
			Bundle: &bundle.Bundle{
				Version:       "1.0.0",
				SchemaVersion: "1.0.0",
				Name:          "Foo App",
			},
		},
		{
			Bundle: &bundle.Bundle{
				Name: "Quiet App",
			},
		},
	}

	testCases := []struct {
		name           string
		expectedOutput string
		options        imageListOption
	}{
		{
			name: "TestList",
			expectedOutput: `REPOSITORY          TAG                 APP IMAGE ID        APP NAME            CREATED             
foo/bar             <none>              3f825b2d0657        Digested App        N/A                 
foo/bar             1.0                 9aae408ee04f        Foo App             N/A                 
<none>              <none>              a855ac937f2e        Quiet App           N/A                 
`,
			options: imageListOption{format: "table"},
		},
		{
			name: "TestTemplate",
			expectedOutput: `APP IMAGE ID        DIGEST
3f825b2d0657        sha256:b59492bb814012ca3d2ce0b6728242d96b4af41687cc82166a4b5d7f2d9fb865
9aae408ee04f        <none>
a855ac937f2e        sha256:a855ac937f2ed375ba4396bbc49c4093e124da933acd2713fb9bc17d7562a087
`,
			options: imageListOption{format: "table {{.ID}}", digests: true},
		},
		{
			name: "TestListWithDigests",
			//nolint:lll
			expectedOutput: `REPOSITORY          TAG                 DIGEST                                                                    APP IMAGE ID        APP NAME                                CREATED             
foo/bar             <none>              sha256:b59492bb814012ca3d2ce0b6728242d96b4af41687cc82166a4b5d7f2d9fb865   3f825b2d0657        Digested App                            N/A                 
foo/bar             1.0                 <none>                                                                    9aae408ee04f        Foo App                                 N/A                 
<none>              <none>              sha256:a855ac937f2ed375ba4396bbc49c4093e124da933acd2713fb9bc17d7562a087   a855ac937f2e        Quiet App                               N/A                 
`,
			options: imageListOption{format: "table", digests: true},
		},
		{
			name: "TestListWithQuiet",
			expectedOutput: `3f825b2d0657
9aae408ee04f
a855ac937f2e
`,
			options: imageListOption{format: "table", quiet: true},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testRunList(t, refs, bundles, tc.options, tc.expectedOutput)
		})
	}
}

func TestSortImages(t *testing.T) {
	images := []imageDesc{
		{ID: "1", Created: time.Date(2016, time.August, 15, 0, 0, 0, 0, time.UTC)},
		{ID: "2"},
		{ID: "3"},
		{ID: "4", Created: time.Date(2018, time.August, 15, 0, 0, 0, 0, time.UTC)},
		{ID: "5", Created: time.Date(2017, time.August, 15, 0, 0, 0, 0, time.UTC)},
	}
	sortImages(images)
	assert.Equal(t, "4", images[0].ID)
	assert.Equal(t, "5", images[1].ID)
	assert.Equal(t, "1", images[2].ID)
	assert.Equal(t, "2", images[3].ID)
	assert.Equal(t, "3", images[4].ID)
}

func parseReference(t *testing.T, s string) reference.Reference {
	ref, err := reference.Parse(s)
	assert.NilError(t, err)
	return ref
}

func testRunList(t *testing.T, refs []reference.Reference, bundles []relocated.Bundle, options imageListOption, expectedOutput string) {
	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)
	dockerCli, err := command.NewDockerCli(command.WithOutputStream(w))
	assert.NilError(t, err)
	bundleStore := &bundleStoreStubForListCmd{
		refMap:  make(map[reference.Reference]*relocated.Bundle),
		refList: []reference.Reference{},
	}
	for i, ref := range refs {
		_, err = bundleStore.Store(ref, &bundles[i])
		assert.NilError(t, err)
	}
	err = runList(dockerCli, options, bundleStore)
	assert.NilError(t, err)
	w.Flush()
	assert.Equal(t, buf.String(), expectedOutput)
}
