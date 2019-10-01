package image

import (
	"fmt"
	"testing"

	"gotest.tools/assert"

	"github.com/deislabs/cnab-go/bundle"
	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/distribution/reference"
)

type bundleStoreStub struct {
	ReadBundle   *bundle.Bundle
	ReadError    error
	StoredBundle string
	StoredError  error
}

func (b *bundleStoreStub) Store(ref reference.Named, bndle *bundle.Bundle) error {
	defer func() {
		b.StoredError = nil
	}()

	b.StoredBundle = ref.String()

	return b.StoredError
}

func (b *bundleStoreStub) Read(ref reference.Named) (*bundle.Bundle, error) {
	defer func() {
		b.ReadBundle = nil
		b.ReadError = nil
	}()
	return b.ReadBundle, b.ReadError
}

func (b *bundleStoreStub) List() ([]reference.Named, error) {
	return nil, nil
}

func (b *bundleStoreStub) Remove(ref reference.Named) error {
	return nil
}

func (b *bundleStoreStub) LookupOrPullBundle(ref reference.Named, pullRef bool, config *configfile.ConfigFile, insecureRegistries []string) (*bundle.Bundle, error) {
	return nil, nil
}

var mockedBundleStore = &bundleStoreStub{}

func TestInvalidSourceReference(t *testing.T) {
	// given a bad source image reference
	const badRef = "b@d reference"

	err := runTag(mockedBundleStore, badRef, "")

	assert.ErrorContains(t, err, fmt.Sprintf("could not parse '%s' as a valid reference", badRef))
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
	// given a bundle is returned by bundleStore.Read
	mockedBundleStore.ReadBundle = &bundle.Bundle{}
	// and given a bad destination reference
	const badRef = "b@d reference"

	err := runTag(mockedBundleStore, "ref", badRef)

	assert.ErrorContains(t, err, fmt.Sprintf("could not parse '%s' as a valid reference", badRef))
}

func TestBundleNotStored(t *testing.T) {
	// given a bundle is returned by bundleStore.Read
	mockedBundleStore.ReadBundle = &bundle.Bundle{}
	// and given bundleStore.Store will return an error
	mockedBundleStore.StoredError = fmt.Errorf("error from bundleStore.Store")

	err := runTag(mockedBundleStore, "src-app", "dest-app")

	assert.Assert(t, err != nil)
}

func TestSuccessfulyTag(t *testing.T) {
	// given a bundle is returned by bundleStore.Read
	mockedBundleStore.ReadBundle = &bundle.Bundle{}
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
