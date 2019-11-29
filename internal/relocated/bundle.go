package relocated

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/opencontainers/go-digest"

	"github.com/pkg/errors"

	"github.com/deislabs/cnab-go/bundle"
	"github.com/docker/cnab-to-oci/relocation"
	"github.com/docker/go/canonical/json"
)

type Bundle struct {
	*bundle.Bundle
	RelocationMap relocation.ImageRelocationMap
	RepoDigest    digest.Digest
}

const (
	BundleFilename        = "bundle.json"
	RelocationMapFilename = "relocation-map.json"
	DigestFilename        = "digest"
)

// FromBundle returns a RelocatedBundle with an empty relocation map.
func FromBundle(bndl *bundle.Bundle) *Bundle {
	return &Bundle{
		Bundle:        bndl,
		RelocationMap: relocation.ImageRelocationMap{},
	}
}

// BundleFromFile creates a relocated bundle based on the bundle file and relocation map.
func BundleFromFile(filename string) (*Bundle, error) {
	bndl, err := BundleJSON(filename)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read bundle")
	}

	relocationMapFileName := filepath.Join(filepath.Dir(filename), RelocationMapFilename)
	relocationMap, err := RelocationMapJSON(relocationMapFileName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read relocation map")
	}

	digestFileName := filepath.Join(filepath.Dir(filename), DigestFilename)
	dg, err := repoDigest(digestFileName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read digest file")
	}

	return &Bundle{
		Bundle:        bndl,
		RelocationMap: relocationMap,
		RepoDigest:    dg,
	}, nil
}

// writeRelocationMap serializes the relocation map and writes it to a file as JSON.
func (b *Bundle) writeRelocationMap(dest string, mode os.FileMode) error {
	d, err := json.MarshalCanonical(b.RelocationMap)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(dest, d, mode)
}

// writeRepoDigest store the repo digest to a file as plain text.
func (b *Bundle) writeRepoDigest(dest string, mode os.FileMode) error {
	if b.RepoDigest == "" {
		return cleanRepoDigest(dest)
	}
	return ioutil.WriteFile(dest, []byte(b.RepoDigest), mode)
}

func cleanRepoDigest(dest string) error {
	if _, err := os.Stat(dest); os.IsNotExist(err) {
		return nil
	}
	return os.Remove(dest)
}

// Store a bundle with the relocation map as json files.
func (b *Bundle) Store(dir string) error {
	// store bundle.json
	path := filepath.Join(dir, BundleFilename)
	if err := b.WriteFile(path, 0644); err != nil {
		return errors.Wrapf(err, "failed to store bundle")
	}

	// store relocation map
	relocationMapPath := filepath.Join(dir, RelocationMapFilename)
	if err := b.writeRelocationMap(relocationMapPath, 0644); err != nil {
		return errors.Wrapf(err, "failed to store relocation map")
	}

	// store repo digest
	repoDigestPath := filepath.Join(dir, DigestFilename)
	if err := b.writeRepoDigest(repoDigestPath, 0644); err != nil {
		return errors.Wrapf(err, "failed to store digest")
	}

	return nil
}

func BundleJSON(bundlePath string) (*bundle.Bundle, error) {
	data, err := ioutil.ReadFile(bundlePath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read file %s", bundlePath)
	}
	bndl, err := bundle.Unmarshal(data)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal file %s", bundlePath)
	}
	return bndl, nil
}

func RelocationMapJSON(relocationMapPath string) (relocation.ImageRelocationMap, error) {
	relocationMap := relocation.ImageRelocationMap{}
	_, err := os.Stat(relocationMapPath)
	if os.IsNotExist(err) {
		// it's ok to not have a relocation map, just act as if the file were empty
		return relocationMap, nil
	}
	data, err := ioutil.ReadFile(relocationMapPath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read file %s", relocationMapPath)
	}
	if err := json.Unmarshal(data, &relocationMap); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal file %s", relocationMapPath)
	}
	return relocationMap, nil
}

func (b *Bundle) RelocatedImages() map[string]bundle.Image {
	images := b.Images
	for name, def := range images {
		if img, ok := b.RelocationMap[def.Image]; ok {
			def.Image = img
			images[name] = def
		}
	}

	return images
}

func repoDigest(digestFileName string) (digest.Digest, error) {
	_, err := os.Stat(digestFileName)
	if os.IsNotExist(err) {
		return "", nil
	}
	bytes, err := ioutil.ReadFile(digestFileName)
	if err != nil {
		return "", err
	}
	return digest.Parse(string(bytes))
}
