package repo

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"os"

	"github.com/Masterminds/semver"
	"github.com/docker/distribution/reference"
)

var (
	// ErrNoAPIVersion indicates that an API version was not specified.
	ErrNoAPIVersion = errors.New("no API version specified")
	// ErrNoBundleVersion indicates that a bundle with the given version is not found.
	ErrNoBundleVersion = errors.New("no bundle with the given version found")
	// ErrNoBundleName indicates that a bundle with the given name is not found.
	ErrNoBundleName = errors.New("no bundle name found")
)

// Index defines a list of bundle repositories, each repository's respective tags and the digest reference.
type Index map[string]map[string]string

// LoadIndex takes a file at the given path and returns an Index object
func LoadIndex(path string) (Index, error) {
	f, err := os.OpenFile(path, os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return loadIndex(f)
}

// LoadIndexReader takes a reader and returns an Index object
func LoadIndexReader(r io.Reader) (Index, error) {
	return loadIndex(r)
}

// LoadIndexBuffer reads repository metadata from a JSON byte stream
func LoadIndexBuffer(data []byte) (Index, error) {
	return loadIndex(bytes.NewBuffer(data))
}

// Add adds a new entry to the index
func (i Index) Add(ref reference.NamedTagged, digest string) {
	if tags, ok := i[reference.FamiliarName(ref)]; ok {
		tags[ref.Tag()] = digest
	} else {
		i[reference.FamiliarName(ref)] = map[string]string{
			ref.Tag(): digest,
		}
	}
}

// Delete removes a bundle from the index.
//
// Returns false if no record was found to delete.
func (i Index) DeleteAll(ref reference.Named) bool {
	_, ok := i[reference.FamiliarName(ref)]
	if ok {
		delete(i, reference.FamiliarName(ref))
	}
	return ok
}

// DeleteVersion removes a single version of a given bundle from the index.
//
// Returns false if the name or version is not found.
func (i Index) DeleteVersion(ref reference.NamedTagged) bool {
	sub, ok := i[reference.FamiliarName(ref)]
	if !ok {
		return false
	}
	_, ok = sub[ref.Tag()]
	if ok {
		delete(sub, ref.Tag())
	}
	return ok
}

// Has returns true if the index has an entry for a bundle with the given name and exact version.
func (i Index) Has(ref reference.NamedTagged) bool {
	_, err := i.GetExactly(ref)
	return err == nil
}

// Get returns the digest for the given name.
//
// If version is empty, this will return the digest for the bundle with the highest version.
func (i Index) Get(ref reference.Named, versionQuery string) (string, error) {
	vs, ok := i[reference.FamiliarName(ref)]
	if !ok {
		return "", ErrNoBundleName
	}
	if len(vs) == 0 {
		return "", ErrNoBundleVersion
	}

	var constraint *semver.Constraints
	if len(versionQuery) == 0 {
		constraint, _ = semver.NewConstraint("*")
	} else {
		var err error
		constraint, err = semver.NewConstraint(versionQuery)
		if err != nil {
			return "", err
		}
	}

	for ver, digest := range vs {
		test, err := semver.NewVersion(ver)
		if err != nil {
			continue
		}

		if constraint.Check(test) {
			return digest, nil
		}
	}
	return "", ErrNoBundleVersion
}

// GetExactly returns the hash of the exact specified version
func (i Index) GetExactly(ref reference.NamedTagged) (string, error) {
	vs, ok := i[reference.FamiliarName(ref)]
	if !ok {
		return "", ErrNoBundleName
	}
	v, ok := vs[ref.Tag()]
	if !ok {
		return "", ErrNoBundleVersion
	}
	return v, nil
}

// GetVersions gets all of the versions for the given name.
//
// The versions are returned as hash keys, where the values are the SHAs
//
// If the name is not found, this will return false.
func (i Index) GetVersions(ref reference.Named) (map[string]string, bool) {
	ret, ok := i[reference.FamiliarName(ref)]
	return ret, ok
}

// WriteFile writes an index file to the given destination path.
//
// The mode on the file is set to 'mode'.
func (i Index) WriteFile(dest string, mode os.FileMode) error {
	b, err := json.MarshalIndent(i, "", "    ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(dest, b, mode)
}

// Merge merges the src index into i (dest).
//
// This merges by name and version.
//
// If one of the entries in the destination index does _not_ already exist, it is added.
// In all other cases, the existing record is preserved.
func (i *Index) Merge(src Index) error {
	for name, versionMap := range src {
		for version, digest := range versionMap {
			named, err := reference.ParseNamed(name)
			if err != nil {
				return err
			}
			versioned, err := reference.WithTag(named, version)
			if err != nil {
				return err
			}
			if !i.Has(versioned) {
				i.Add(versioned, digest)
			}
		}
	}
	return nil
}

// loadIndex loads an index file and does minimal validity checking.
func loadIndex(r io.Reader) (Index, error) {
	i := Index{}
	if err := json.NewDecoder(r).Decode(&i); err != nil && err != io.EOF {
		return i, err
	}
	return i, nil
}
