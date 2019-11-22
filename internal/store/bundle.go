package store

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/docker/app/internal/relocated"

	"github.com/docker/distribution/reference"
	digest "github.com/opencontainers/go-digest"
	"github.com/pkg/errors"
)

//
type BundleStore interface {
	// Store do store the bundle with optional reference, and return it's unique ID
	Store(ref reference.Reference, bndl *relocated.Bundle) (reference.Digested, error)
	Read(ref reference.Reference) (*relocated.Bundle, error)
	List() ([]reference.Reference, error)
	Remove(ref reference.Reference, force bool) error
	LookUp(refOrID string) (reference.Reference, error)
}

var _ BundleStore = &bundleStore{}

type referencesMap map[ID][]reference.Reference

type bundleStore struct {
	path    string
	refsMap referencesMap
}

// NewBundleStore creates a new bundle store with the given path and initializes it
func NewBundleStore(path string) (BundleStore, error) {
	bundleStore := &bundleStore{
		path:    path,
		refsMap: make(referencesMap),
	}
	if err := bundleStore.scanAllBundles(); err != nil {
		return nil, err
	}
	return bundleStore, nil
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

func (b *bundleStore) Store(ref reference.Reference, bndl *relocated.Bundle) (reference.Digested, error) {
	id, err := FromBundle(bndl)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to store bundle %q", ref)
	}
	if ref == nil {
		ref = id
	}
	dir, err := b.storePath(ref)
	if err != nil {
		return id, errors.Wrapf(err, "failed to store bundle %q", ref)
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return id, errors.Wrapf(err, "failed to store bundle %q", ref)
	}

	if err := bndl.Store(dir); err != nil {
		return id, errors.Wrapf(err, "failed to store relocated bundle %q", ref)
	}

	b.refsMap.appendRef(id, ref)
	return id, nil
}

func (b *bundleStore) Read(ref reference.Reference) (*relocated.Bundle, error) {
	paths, err := b.storePaths(ref)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read bundle %q", ref)
	}

	return relocated.BundleFromFile(filepath.Join(paths[0], relocated.BundleFilename))
}

// Returns the list of all bundles present in the bundle store
func (b *bundleStore) List() ([]reference.Reference, error) {
	var references []reference.Reference

	for _, refAliases := range b.refsMap {
		references = append(references, refAliases...)
	}

	sort.Slice(references, func(i, j int) bool {
		return references[i].String() < references[j].String()
	})

	return references, nil
}

// Remove removes a bundle from the bundle store.
func (b *bundleStore) Remove(ref reference.Reference, force bool) error {
	if id, ok := ref.(ID); ok {
		refs := b.refsMap[id]
		if len(refs) == 0 {
			return fmt.Errorf("no such image %q", reference.FamiliarString(ref))
		} else if len(refs) > 1 {
			var failure error
			if force {
				toDelete := append([]reference.Reference{}, refs...)
				for _, r := range toDelete {
					if err := b.doRemove(r); err != nil {
						failure = err
					}
				}
				return failure
			}
			return fmt.Errorf("unable to delete %q - App is referenced in multiple repositories", reference.FamiliarString(ref))
		}
		ref = refs[0]
	}
	return b.doRemove(ref)
}

func (b *bundleStore) doRemove(ref reference.Reference) error {
	path, err := b.storePath(ref)
	if err != nil {
		return err
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return errors.New("no such image " + reference.FamiliarString(ref))
	}
	b.refsMap.removeRef(ref)

	if err := os.RemoveAll(path); err != nil {
		return nil
	}
	return cleanupParentTree(path)
}

func cleanupParentTree(path string) error {
	for {
		path = filepath.Dir(path)
		if empty, err := isEmpty(path); err != nil || !empty {
			return err
		}
		if err := os.RemoveAll(path); err != nil {
			return nil
		}
	}
}

func isEmpty(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()
	if _, err = f.Readdir(1); err == io.EOF {
		// dir is empty
		return true, nil
	}
	return false, nil
}

func (b *bundleStore) LookUp(refOrID string) (reference.Reference, error) {
	ref, err := FromString(refOrID)
	if err == nil {
		if _, found := b.refsMap[ref]; !found {
			return nil, unknownReference(refOrID)
		}
		return ref, nil
	}
	if isShortID(refOrID) {
		ref, err := b.matchShortID(refOrID)
		if err == nil {
			return ref, nil
		}
	}
	named, err := StringToNamedRef(refOrID)
	if err != nil {
		return nil, err
	}
	if _, err = b.referenceToID(named); err != nil {
		return nil, err
	}
	return named, nil
}

func (b *bundleStore) matchShortID(shortID string) (reference.Reference, error) {
	var found reference.Reference
	for id := range b.refsMap {
		if strings.HasPrefix(id.String(), shortID) {
			if found != nil && found != id {
				return nil, fmt.Errorf("ambiguous reference found")
			}
			found = id
		}
	}
	if found == nil {
		return nil, unknownReference(shortID)
	}
	return found, nil
}

func (b *bundleStore) referenceToID(ref reference.Reference) (ID, error) {
	if id, ok := ref.(ID); ok {
		return id, nil
	}
	for id, refs := range b.refsMap {
		for _, r := range refs {
			if r == ref {
				return id, nil
			}
		}
	}
	return ID{}, unknownReference(reference.FamiliarString(ref))
}

func (b *bundleStore) storePaths(ref reference.Reference) ([]string, error) {
	var paths []string

	id, err := b.referenceToID(ref)
	if err != nil {
		return nil, err
	}

	if refs, exist := b.refsMap[id]; exist {
		for _, rf := range refs {
			path, err := b.storePath(rf)
			if err != nil {
				return nil, err
			}
			paths = append(paths, path)
		}
	}

	if len(paths) == 0 {
		return nil, unknownReference(reference.FamiliarString(ref))
	}
	return paths, nil
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

// scanAllBundles scans the bundle store directories and creates the internal map of App image
// references. This function must be called before any other public BundleStore interface method.
func (b *bundleStore) scanAllBundles() error {
	if err := filepath.Walk(b.path, b.processBundleStoreFile); err != nil {
		return err
	}
	return nil
}

func (b *bundleStore) processBundleStoreFile(path string, info os.FileInfo, err error) error {
	if err != nil {
		return err
	}
	idRefPath := filepath.Join(b.path, "_ids")

	if info.IsDir() {
		return nil
	}

	if info.Name() == relocated.RelocationMapFilename {
		return nil
	}

	if !strings.HasSuffix(info.Name(), ".json") {
		return nil
	}

	if strings.HasPrefix(path, idRefPath) {
		rel := path[len(idRefPath)+1:]
		dg := strings.Split(filepath.ToSlash(rel), "/")[0]
		id := ID{digest.NewDigestFromEncoded(digest.SHA256, dg)}
		b.refsMap.appendRef(id, id)
		return nil
	}

	ref, err := b.pathToReference(path)
	if err != nil {
		return err
	}
	bndl, err := relocated.BundleFromFile(path)
	if err != nil {
		return err
	}
	id, err := FromBundle(bndl)
	if err != nil {
		return err
	}
	b.refsMap[id] = append(b.refsMap[id], ref)

	return nil
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

func (rm referencesMap) appendRef(id ID, ref reference.Reference) {
	if _, found := rm[id]; found {
		if !containsRef(rm[id], ref) {
			rm[id] = append(rm[id], ref)
		}
	} else {
		rm[id] = []reference.Reference{ref}
	}
}

func (rm referencesMap) removeRef(ref reference.Reference) {
	for id, refs := range rm {
		for i, r := range refs {
			if r == ref {
				rm[id] = append(refs[:i], refs[i+1:]...)
				if len(rm[id]) == 0 {
					delete(rm, id)
				}
				return
			}
		}
	}
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

func containsRef(list []reference.Reference, ref reference.Reference) bool {
	for _, v := range list {
		if v == ref {
			return true
		}
	}
	return false
}

func unknownReference(ref string) *UnknownReferenceError {
	return &UnknownReferenceError{ref}
}

// UnknownReferenceError represents a reference not found in the bundle store
type UnknownReferenceError struct {
	string
}

func (e *UnknownReferenceError) Error() string {
	return fmt.Sprintf("%s: reference not found", e.string)
}

// NotFound satisfies interface github.com/docker/docker/errdefs.ErrNotFound
func (e *UnknownReferenceError) NotFound() {}
