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
	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)
	dockerCli, err := command.NewDockerCli(command.WithOutputStream(w))
	assert.NilError(t, err)
	bundleStore := &bundleStoreStubForListCmd{
		refMap:  make(map[reference.Reference]*bundle.Bundle),
		refList: []reference.Reference{},
	}
	ref1, err := store.FromString("a855ac937f2ed375ba4396bbc49c4093e124da933acd2713fb9bc17d7562a087")
	assert.NilError(t, err)
	ref2, err := reference.Parse("foo/bar:1.0")
	assert.NilError(t, err)
	_, err = bundleStore.Store(ref1, &bundle.Bundle{})
	assert.NilError(t, err)
	_, err = bundleStore.Store(ref2, &bundle.Bundle{
		Version:       "1.0.0",
		SchemaVersion: "1.0.0",
		Name:          "Foo App",
	})
	assert.NilError(t, err)
	err = runList(dockerCli, imageListOption{quiet: true}, bundleStore)
	assert.NilError(t, err)
	expectedOutput := `a855ac937f2e
9aae408ee04f
`
	w.Flush()
	assert.Equal(t, buf.String(), expectedOutput)
}
