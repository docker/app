package image

import (
	"fmt"
	"testing"

	"github.com/docker/app/internal/image"

	"gotest.tools/assert"

	"github.com/docker/distribution/reference"
)

func parseRefOrDie(t *testing.T, ref string) reference.Named {
	t.Helper()
	named, err := reference.ParseNormalizedNamed(ref)
	assert.NilError(t, err)
	return named
}

type imageStoreStub struct {
	ReadBundle   *image.AppImage
	ReadError    error
	StoredBundle string
	StoredError  error
	StoredID     reference.Digested
	LookUpRef    reference.Reference
	LookUpError  error
}

func (b *imageStoreStub) Store(ref reference.Reference, bndle *image.AppImage) (reference.Digested, error) {
	defer func() {
		b.StoredError = nil
	}()

	b.StoredBundle = ref.String()

	return b.StoredID, b.StoredError
}

func (b *imageStoreStub) Read(ref reference.Reference) (*image.AppImage, error) {
	defer func() {
		b.ReadBundle = nil
		b.ReadError = nil
	}()
	return b.ReadBundle, b.ReadError
}

func (b *imageStoreStub) List() ([]reference.Reference, error) {
	return nil, nil
}

func (b *imageStoreStub) Remove(ref reference.Reference, force bool) error {
	return nil
}

func (b *imageStoreStub) LookUp(refOrID string) (reference.Reference, error) {
	defer func() {
		b.LookUpRef = nil
		b.LookUpError = nil
	}()
	return b.LookUpRef, b.LookUpError
}

var mockedImageStore = &imageStoreStub{}

func TestInvalidSourceReference(t *testing.T) {
	// given a bad source image reference
	const badRef = "b@d reference"
	// and given bundle store will return an error on LookUp
	mockedImageStore.LookUpError = fmt.Errorf("error from imageStore.LookUp")

	err := runTag(mockedImageStore, badRef, "")

	assert.Assert(t, err != nil)
}

func TestUnexistingSource(t *testing.T) {
	// given a well formatted source image reference
	const unexistingRef = "unexisting"
	// and given bundle store will return an error on Read
	mockedImageStore.ReadError = fmt.Errorf("error from imageStore.Read")

	err := runTag(mockedImageStore, unexistingRef, "dest")

	assert.Assert(t, err != nil)
}

func TestInvalidDestinationReference(t *testing.T) {
	// given a reference and a bundle is returned by imageStore.LookUp and imageStore.Read
	mockedImageStore.LookUpRef = parseRefOrDie(t, "ref")
	mockedImageStore.ReadBundle = &image.AppImage{}
	// and given a bad destination reference
	const badRef = "b@d reference"

	err := runTag(mockedImageStore, "ref", badRef)

	assert.ErrorContains(t, err, fmt.Sprintf("invalid reference format"))
}

func TestBundleNotStored(t *testing.T) {
	// given a reference and a bundle is returned by imageStore.LookUp and imageStore.Read
	mockedImageStore.LookUpRef = parseRefOrDie(t, "src-app")
	mockedImageStore.ReadBundle = &image.AppImage{}
	// and given imageStore.Store will return an error
	mockedImageStore.StoredError = fmt.Errorf("error from imageStore.Store")

	err := runTag(mockedImageStore, "src-app", "dest-app")

	assert.Assert(t, err != nil)
}

func TestSuccessfulyTag(t *testing.T) {
	// given a reference and a bundle is returned by imageStore.LookUp and imageStore.Read
	mockedImageStore.LookUpRef = parseRefOrDie(t, "src-app")
	mockedImageStore.ReadBundle = &image.AppImage{}
	// and given valid source and output references
	const (
		srcRef            = "src-app"
		destRef           = "dest-app"
		normalizedDestRef = "docker.io/library/dest-app:latest"
	)

	err := runTag(mockedImageStore, srcRef, destRef)

	assert.NilError(t, err)
	assert.Equal(t, mockedImageStore.StoredBundle, normalizedDestRef)
}
