package store

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/deislabs/cnab-go/bundle"
	"github.com/docker/distribution/reference"
	"github.com/pkg/errors"
)

//
type BundleStore interface {
	Store(ref reference.Reference, bndle *bundle.Bundle) (reference.Reference, error)
	Read(ref reference.Reference) (*bundle.Bundle, error)
	List() ([]reference.Reference, error)
	Remove(ref reference.Reference) error
}

var _ BundleStore = &bundleStore{}

type bundleStore struct {
	path string
}

// We store bundles either by image:tags, image:digest or by unique ID (actually, bundle's sha256).
//
// Within the bundle store, the file layout is
// <registry>
//  \_ <repo>
//       \_ _tags
//           \_ <tag>
//                \_ bundle.json
//       \_ _digests
//            \_ <algorithm>
//                \_ <digested-reference>
//                     \_ bundle.json
// _ids
//  \_ <bundle_id>
//      \_ bundle.json
//

func (b *bundleStore) Store(ref reference.Reference, bndle *bundle.Bundle) (reference.Reference, error) {
	if ref == nil {
		digest, err := ComputeDigest(bndle)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to store bundle %q", ref)
		}
		ref = ID{digest.Encoded()}
	}
	dir, err := b.storePath(ref)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to store bundle %q", ref)
	}
	path := filepath.Join(dir, "bundle.json")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, errors.Wrapf(err, "failed to store bundle %q", ref)
	}
	if err = bndle.WriteFile(path, 0644); err != nil {
		return nil, errors.Wrapf(err, "failed to store bundle %q", ref)
	}
	return ref, nil
}

func (b *bundleStore) Read(ref reference.Reference) (*bundle.Bundle, error) {
	path, err := b.storePath(ref)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read bundle %q", ref)
	}

	data, err := ioutil.ReadFile(filepath.Join(path, "bundle.json"))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read bundle %q", ref)
	}
	var bndle bundle.Bundle
	if err := json.Unmarshal(data, &bndle); err != nil {
		return nil, errors.Wrapf(err, "failed to read bundle %q", ref)
	}
	return &bndle, nil
}

// Returns the list of all bundles present in the bundle store
func (b *bundleStore) List() ([]reference.Reference, error) {
	var references []reference.Reference
	digests := filepath.Join(b.path, "_ids")
	if err := filepath.Walk(b.path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if !strings.HasSuffix(info.Name(), ".json") {
			return nil
		}

		if strings.HasPrefix(path, digests) {
			rel := path[len(digests)+1:]
			dg := strings.Split(filepath.ToSlash(rel), "/")[0]
			references = append(references, ID{dg})
			return nil
		}

		ref, err := b.pathToReference(path)
		if err != nil {
			return err
		}

		references = append(references, ref)

		return nil
	}); err != nil {
		return nil, err
	}

	sort.Slice(references, func(i, j int) bool {
		return references[i].String() < references[j].String()
	})

	return references, nil
}

// Remove removes a bundle from the bundle store.
func (b *bundleStore) Remove(ref reference.Reference) error {
	path, err := b.storePath(ref)
	if err != nil {
		return err
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return errors.New("no such image " + reference.FamiliarString(ref))
	}
	return os.RemoveAll(path)
}

func (b *bundleStore) storePath(ref reference.Reference) (string, error) {
	named, ok := ref.(reference.Named)
	if !ok {
		return filepath.Join(b.path, "_ids", ref.String()), nil
	}

	name := strings.Replace(named.Name(), ":", "_", 1)
	// A name is safe for use as a filesystem path (it is
	// alphanumerics + "." + "/") except for the ":" used to
	// separate domain from port which is not safe on Windows.
	// Replace it with "_" which is not valid in the name.
	//
	// There can be at most 1 ":" in a valid reference so only
	// replace one -- if there are more (and this wasn't caught
	// when parsing the ref) then there will be errors when we try
	// to use this as a path later.
	storeDir := filepath.Join(b.path, filepath.FromSlash(name))

	// We rely here on _ not being valid in a name meaning there can be no clashes due to nesting of repositories.
	switch t := ref.(type) {
	case reference.Digested:
		digest := t.Digest()
		storeDir = filepath.Join(storeDir, "_digests", digest.Algorithm().String(), digest.Encoded())
	case reference.Tagged:
		storeDir = filepath.Join(storeDir, "_tags", t.Tag())
	default:
		return "", errors.Errorf("%s: not tagged or digested", ref.String())
	}

	return storeDir, nil
}

func (b *bundleStore) pathToReference(path string) (reference.Named, error) {
	// Clean the path and remove the local bundle store path
	cleanpath := filepath.ToSlash(path)
	cleanpath = strings.TrimPrefix(cleanpath, filepath.ToSlash(b.path)+"/")

	// get the hierarchy of directories, so we can get digest algorithm or tag
	paths := strings.Split(cleanpath, "/")
	if len(paths) < 3 {
		return nil, fmt.Errorf("invalid path %q in the bundle store", path)
	}

	// path must point to a json file
	if !strings.Contains(paths[len(paths)-1], ".json") {
		return nil, fmt.Errorf("invalid path %q, not referencing a CNAB bundle in json format", path)
	}

	// remove the bundle.json filename
	paths = paths[:len(paths)-1]

	name, err := reconstructNamedReference(path, paths)
	if err != nil {
		return nil, err
	}

	return reference.ParseNamed(name)
}

func reconstructNamedReference(path string, paths []string) (string, error) {
	name, paths := strings.Replace(paths[0], "_", ":", 1), paths[1:]
	for i, p := range paths {
		switch p {
		case "_tags":
			if i != len(paths)-2 {
				return "", fmt.Errorf("invalid path %q in the bundle store", path)
			}
			return fmt.Sprintf("%s:%s", name, paths[i+1]), nil
		case "_digests":
			if i != len(paths)-3 {
				return "", fmt.Errorf("invalid path %q in the bundle store", path)
			}
			return fmt.Sprintf("%s@%s:%s", name, paths[i+1], paths[i+2]), nil
		default:
			name += "/" + p
		}
	}
	return name, nil
}
