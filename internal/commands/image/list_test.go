package image

import (
	"bufio"
	"bytes"
	"fmt"
	"testing"

	"gotest.tools/assert"

	"github.com/deislabs/cnab-go/bundle"
	"github.com/docker/app/internal/store"
	"github.com/docker/cli/cli/command"
	"github.com/docker/distribution/reference"
)

type bundleStoreStubForListCmd struct {
	refMap map[reference.Reference]*bundle.Bundle
	// in order to keep the reference in the same order between tests
	refList []reference.Reference
}

func (b *bundleStoreStubForListCmd) Store(ref reference.Reference, bndle *bundle.Bundle) (reference.Digested, error) {
	b.refMap[ref] = bndle
	b.refList = append(b.refList, ref)
	return store.FromBundle(bndle)
}

func (b *bundleStoreStubForListCmd) Read(ref reference.Reference) (*bundle.Bundle, error) {
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

func TestListWithQuietFlag(t *testing.T) {
	ref, err := store.FromString("a855ac937f2ed375ba4396bbc49c4093e124da933acd2713fb9bc17d7562a087")
	assert.NilError(t, err)
	refs := []reference.Reference{
		ref,
		parseReference(t, "foo/bar:1.0"),
	}
	bundles := []bundle.Bundle{
		{},
		{
			Version:       "1.0.0",
			SchemaVersion: "1.0.0",
			Name:          "Foo App",
		},
	}
	expectedOutput := `a855ac937f2e
9aae408ee04f
`
	testRunList(t, refs, bundles, imageListOption{quiet: true}, expectedOutput)
}

func TestListWithDigestsFlag(t *testing.T) {
	refs := []reference.Reference{
		parseReference(t, "foo/bar@sha256:b59492bb814012ca3d2ce0b6728242d96b4af41687cc82166a4b5d7f2d9fb865"),
		parseReference(t, "foo/bar:1.0"),
	}
	bundles := []bundle.Bundle{
		{
			Name: "Digested App",
		},
		{
			Version:       "1.0.0",
			SchemaVersion: "1.0.0",
			Name:          "Foo App",
		},
	}
	expectedOutput := `APP IMAGE                                                                       DIGEST                                                                  APP NAME
foo/bar@sha256:b59492bb814012ca3d2ce0b6728242d96b4af41687cc82166a4b5d7f2d9fb865 sha256:b59492bb814012ca3d2ce0b6728242d96b4af41687cc82166a4b5d7f2d9fb865 Digested App
foo/bar:1.0                                                                     <none>                                                                  Foo App
`
	testRunList(t, refs, bundles, imageListOption{digests: true}, expectedOutput)
}

func parseReference(t *testing.T, s string) reference.Reference {
	ref, err := reference.Parse(s)
	assert.NilError(t, err)
	return ref
}

func testRunList(t *testing.T, refs []reference.Reference, bundles []bundle.Bundle, options imageListOption, expectedOutput string) {
	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)
	dockerCli, err := command.NewDockerCli(command.WithOutputStream(w))
	assert.NilError(t, err)
	bundleStore := &bundleStoreStubForListCmd{
		refMap:  make(map[reference.Reference]*bundle.Bundle),
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
