package image

import (
	"fmt"
	"testing"

	"github.com/docker/app/internal/relocated"

	"gotest.tools/assert"

	"github.com/docker/distribution/reference"
)

func parseRefOrDie(t *testing.T, ref string) reference.Named {
	t.Helper()
	named, err := reference.ParseNormalizedNamed(ref)
	assert.NilError(t, err)
	return named
}

type bundleStoreStub struct {
	ReadBundle   *relocated.Bundle
	ReadError    error
	StoredBundle string
	StoredError  error
	StoredID     reference.Digested
	LookUpRef    reference.Reference
	LookUpError  error
}

func (b *bundleStoreStub) Store(ref reference.Reference, bndle *relocated.Bundle) (reference.Digested, error) {
	defer func() {
		b.StoredError = nil
	}()

	b.StoredBundle = ref.String()

	return b.StoredID, b.StoredError
}

func (b *bundleStoreStub) Read(ref reference.Reference) (*relocated.Bundle, error) {
	defer func() {
		b.ReadBundle = nil
		b.ReadError = nil
	}()
	return b.ReadBundle, b.ReadError
}

func (b *bundleStoreStub) List() ([]reference.Reference, error) {
	return nil, nil
}

func (b *bundleStoreStub) Remove(ref reference.Reference) error {
	return nil
}

func (b *bundleStoreStub) LookUp(refOrID string) (reference.Reference, error) {
	defer func() {
		b.LookUpRef = nil
		b.LookUpError = nil
	}()
	return b.LookUpRef, b.LookUpError
}

var mockedBundleStore = &bundleStoreStub{}

func TestInvalidSourceReference(t *testing.T) {
	// given a bad source image reference
	const badRef = "b@d reference"
	// and given bundle store will return an error on LookUp
	mockedBundleStore.LookUpError = fmt.Errorf("error from bundleStore.LookUp")

	err := runTag(mockedBundleStore, badRef, "")

	assert.Assert(t, err != nil)
}

func TestUnexistingSource(t *testing.T) {
	// given a well formatted source image reference
	const unexistingRef = "unexisting"
	// and given bundle store will return an error on Read
	mockedBundleStore.ReadError = fmt.Errorf("error from bundleStore.Read")

	err := runTag(mockedBundleStore, unexistingRef, "dest")

	assert.Assert(t, err != nil)
}

func TestInvalidDestinationReference(t *testing.T) {
	// given a reference and a bundle is returned by bundleStore.LookUp and bundleStore.Read
	mockedBundleStore.LookUpRef = parseRefOrDie(t, "ref")
	mockedBundleStore.ReadBundle = &relocated.Bundle{}
	// and given a bad destination reference
	const badRef = "b@d reference"

	err := runTag(mockedBundleStore, "ref", badRef)

	assert.ErrorContains(t, err, fmt.Sprintf("invalid reference format"))
}

func TestBundleNotStored(t *testing.T) {
	// given a reference and a bundle is returned by bundleStore.LookUp and bundleStore.Read
	mockedBundleStore.LookUpRef = parseRefOrDie(t, "src-app")
	mockedBundleStore.ReadBundle = &relocated.Bundle{}
	// and given bundleStore.Store will return an error
	mockedBundleStore.StoredError = fmt.Errorf("error from bundleStore.Store")

	err := runTag(mockedBundleStore, "src-app", "dest-app")

	assert.Assert(t, err != nil)
}

func TestSuccessfulyTag(t *testing.T) {
	// given a reference and a bundle is returned by bundleStore.LookUp and bundleStore.Read
	mockedBundleStore.LookUpRef = parseRefOrDie(t, "src-app")
	mockedBundleStore.ReadBundle = &relocated.Bundle{}
	// and given valid source and output references
	const (
		srcRef            = "src-app"
		destRef           = "dest-app"
		normalizedDestRef = "docker.io/library/dest-app:latest"
	)

	err := runTag(mockedBundleStore, srcRef, destRef)

	assert.NilError(t, err)
	assert.Equal(t, mockedBundleStore.StoredBundle, normalizedDestRef)
}
