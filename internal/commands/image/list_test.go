package image

import (
	"bufio"
	"bytes"
	"fmt"
	"testing"

	"github.com/docker/app/internal/image"

	"gotest.tools/assert"

	"github.com/deislabs/cnab-go/bundle"
	"github.com/docker/app/internal/store"
	"github.com/docker/cli/cli/command"
	"github.com/docker/distribution/reference"
)

type imageStoreStubForListCmd struct {
	refMap map[reference.Reference]*image.AppImage
	// in order to keep the reference in the same order between tests
	refList []reference.Reference
}

func (b *imageStoreStubForListCmd) Store(bndl *image.AppImage, ref reference.Reference) (reference.Digested, error) {
	b.refMap[ref] = bndl
	b.refList = append(b.refList, ref)
	return store.FromAppImage(bndl)
}

func (b *imageStoreStubForListCmd) Read(ref reference.Reference) (*image.AppImage, error) {
	bndl, ok := b.refMap[ref]
	if ok {
		return bndl, nil
	}
	return nil, fmt.Errorf("AppImage not found")
}

func (b *imageStoreStubForListCmd) List() ([]reference.Reference, error) {
	return b.refList, nil
}

func (b *imageStoreStubForListCmd) Remove(ref reference.Reference, force bool) error {
	return nil
}

func (b *imageStoreStubForListCmd) LookUp(refOrID string) (reference.Reference, error) {
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
	bundles := []image.AppImage{
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
			expectedOutput: `REPOSITORY          TAG                 APP IMAGE ID        APP NAME            
foo/bar             <none>              3f825b2d0657        Digested App        
foo/bar             1.0                 9aae408ee04f        Foo App             
<none>              <none>              a855ac937f2e        Quiet App           
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
			expectedOutput: `REPOSITORY          TAG                 DIGEST                                                                    APP IMAGE ID        APP NAME                                
foo/bar             <none>              sha256:b59492bb814012ca3d2ce0b6728242d96b4af41687cc82166a4b5d7f2d9fb865   3f825b2d0657        Digested App                            
foo/bar             1.0                 <none>                                                                    9aae408ee04f        Foo App                                 
<none>              <none>              sha256:a855ac937f2ed375ba4396bbc49c4093e124da933acd2713fb9bc17d7562a087   a855ac937f2e        Quiet App                               
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

func parseReference(t *testing.T, s string) reference.Reference {
	ref, err := reference.Parse(s)
	assert.NilError(t, err)
	return ref
}

func testRunList(t *testing.T, refs []reference.Reference, bundles []image.AppImage, options imageListOption, expectedOutput string) {
	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)
	dockerCli, err := command.NewDockerCli(command.WithOutputStream(w))
	assert.NilError(t, err)
	imageStore := &imageStoreStubForListCmd{
		refMap:  make(map[reference.Reference]*image.AppImage),
		refList: []reference.Reference{},
	}
	for i, ref := range refs {
		_, err = imageStore.Store(&bundles[i], ref)
		assert.NilError(t, err)
	}
	err = runList(dockerCli, options, imageStore)
	assert.NilError(t, err)
	w.Flush()
	assert.Equal(t, buf.String(), expectedOutput)
}
