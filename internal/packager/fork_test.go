package packager

import (
	"testing"

	"github.com/docker/app/types/metadata"
	"gotest.tools/assert"
)

func TestSplitPackageName(t *testing.T) {
	ns, name := splitPackageName("foo/bar")
	assert.Equal(t, ns, "foo")
	assert.Equal(t, name, "bar")

	ns, name = splitPackageName("nonamespace")
	assert.Equal(t, ns, "")
	assert.Equal(t, name, "nonamespace")

	ns, name = splitPackageName("some.repo.tk/v3/foo/bar")
	assert.Equal(t, ns, "some.repo.tk/v3/foo")
	assert.Equal(t, name, "bar")
}

var sampleMetadata = `
name: machine
namespace: heavy.metal
version: "2000"
maintainers:
  - name: Billy Corgan
    email: billy@pumpkins.net
`

var decodedSampleMetadata = metadata.AppMetadata{
	Name:      "machine",
	Namespace: "heavy.metal",
	Version:   "2000",
	Maintainers: []metadata.Maintainer{
		{Name: "Billy Corgan", Email: "billy@pumpkins.net"},
	},
}

func TestLoadMetadata(t *testing.T) {
	appmeta, err := loadMetadata([]byte(sampleMetadata))
	assert.NilError(t, err)
	assert.DeepEqual(t, appmeta, metadata.AppMetadata{
		Name:      "machine",
		Namespace: "heavy.metal",
		Version:   "2000",
		Maintainers: []metadata.Maintainer{
			{Name: "Billy Corgan", Email: "billy@pumpkins.net"},
		},
	})
}

func TestLoadEmptyMetadata(t *testing.T) {
	appmeta, err := loadMetadata([]byte(""))
	assert.NilError(t, err)
	assert.DeepEqual(t, appmeta, metadata.AppMetadata{})
}

func TestLoadInvalidMetadata(t *testing.T) {
	_, err := loadMetadata([]byte("'rootstring'"))
	assert.ErrorContains(t, err, "failed to parse application metadata")
}

func TestUpdateMetadata(t *testing.T) {
	newNamespace := "frog"
	newName := "machine"
	maintainers := []string{"infected mushroom:im@psy.net"}

	output, err := updateMetadata([]byte(sampleMetadata), newNamespace, newName, maintainers)
	assert.NilError(t, err)
	decodedOutput, err := loadMetadata(output)
	assert.NilError(t, err)
	assert.DeepEqual(t, decodedOutput, metadata.AppMetadata{
		Name:      "machine",
		Namespace: "frog",
		Version:   decodedSampleMetadata.Version,
		Maintainers: []metadata.Maintainer{
			{Name: "infected mushroom", Email: "im@psy.net"},
		},
		Parents: metadata.Parents{
			{
				Name:        decodedSampleMetadata.Name,
				Namespace:   decodedSampleMetadata.Namespace,
				Version:     decodedSampleMetadata.Version,
				Maintainers: decodedSampleMetadata.Maintainers,
			},
		},
	})
}

func TestUpdateMetadataInvalidOrigin(t *testing.T) {
	_, err := updateMetadata([]byte("'rootstring'"), "", "", []string{})
	assert.ErrorContains(t, err, "failed to parse application metadata")
}
