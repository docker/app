package store

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/docker/app/internal/image"
	"github.com/docker/distribution/reference"
	refstore "github.com/docker/docker/reference"
	"github.com/hashicorp/go-multierror"
	"github.com/opencontainers/go-digest"
	"github.com/pkg/errors"
)

//
type ImageStore interface {
	// Store do store the bundle with optional reference, and return it's unique ID
	Store(img *image.AppImage, ref reference.Reference) (reference.Digested, error)
	Read(ref reference.Reference) (*image.AppImage, error)
	List() ([]reference.Reference, error)
	Remove(ref reference.Reference, force bool) error
	LookUp(refOrID string) (reference.Reference, error)
}

var _ ImageStore = &imageStore{}

type referencesMap map[ID][]reference.Reference

type imageStore struct {
	path    string
	refsMap referencesMap
	store   refstore.Store
}

// NewImageStore creates a new bundle store with the given path and initializes it
func NewImageStore(path string) (ImageStore, error) {
	err := os.MkdirAll(filepath.Join(path, "contents", "sha256"), 0755)
	if err != nil {
		return nil, err
	}
	store, err := refstore.NewReferenceStore(filepath.Join(path, "repositories.json"))
	if err != nil {
		return nil, err
	}
	imageStore := &imageStore{
		path:    path,
		refsMap: make(referencesMap),
		store:   store,
	}
	return imageStore, nil
}

// We store bundles either by image:tags, image:digest or by unique ID (actually, bundle's sha256).
//
// Within the bundle store, the file layout is
// contents
//  \_ <bundle_id>
//      \_ bundle.json
//      \_ relocation-map.json
// repositories.json  // managed by docker/reference
//

func (b *imageStore) Store(img *image.AppImage, ref reference.Reference) (reference.Digested, error) {
	id, err := FromAppImage(img)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to store bundle %q", ref)
	}
	dir := b.storePath(id.Digest())
	if err := os.MkdirAll(dir, 0755); err != nil {
		return id, errors.Wrapf(err, "failed to store bundle %q", ref)
	}

	if err := img.Store(dir); err != nil {
		return id, errors.Wrapf(err, "failed to store app image %q", ref)
	}

	if tag, ok := ref.(reference.NamedTagged); ok {
		err = b.store.AddTag(reference.TagNameOnly(tag), id.Digest(), true)
		if err != nil {
			return nil, err
		}
	}
	if digest, ok := ref.(reference.Canonical); ok {
		err = b.store.AddDigest(digest, id.Digest(), true)
		if err != nil {
			return nil, err
		}
	}

	return id, nil
}

func (b *imageStore) Read(ref reference.Reference) (*image.AppImage, error) {
	var dg digest.Digest
	if id, ok := ref.(ID); ok {
		dg = id.Digest()
	}
	if named, ok := ref.(reference.Named); ok {
		resolved, err := b.store.Get(reference.TagNameOnly(named))
		if err == refstore.ErrDoesNotExist {
			return nil, unknownReference(ref.String())
		}
		if err != nil {
			return nil, err
		}
		dg = resolved
	}
	path := b.storePath(dg)
	return image.FromFile(filepath.Join(path, image.BundleFilename))
}

// Returns the list of all bundles present in the bundle store
func (b *imageStore) List() ([]reference.Reference, error) {
	ids, err := b.listIDs()
	if err != nil {
		return nil, err
	}

	references := []reference.Reference{}
	for _, dg := range ids {
		id := fromID(dg)
		refs := b.store.References(id.Digest())
		for _, r := range refs {
			references = append(references, r)
		}
		if len(refs) == 0 {
			references = append(references, id)
		}
	}

	sort.Slice(references, func(i, j int) bool {
		return references[i].String() < references[j].String()
	})

	return references, nil
}

func (b *imageStore) listIDs() ([]string, error) {
	f, err := os.Open(filepath.Join(b.path, "contents", "sha256"))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	ids, err := f.Readdirnames(-1)
	if err != nil {
		return nil, err
	}
	return ids, nil
}

// Remove removes a bundle from the bundle store.
func (b *imageStore) Remove(ref reference.Reference, force bool) error {
	if named, ok := ref.(reference.Named); ok {
		named = reference.TagNameOnly(named)
		id, err := b.store.Get(named)
		if err != nil {
			return err
		}
		_, err = b.store.Delete(named)
		references := b.store.References(id)
		if len(references) > 0 {
			return err
		}
		// No tag left for ID, so also remove
		ref = ID(id)
	}
	id := ref.(ID)
	refs := b.store.References(id.Digest())
	if len(refs) > 1 && !force {
		return fmt.Errorf("unable to delete %q - App is referenced in multiple repositories", reference.FamiliarString(ref))
	}
	var failures *multierror.Error
	for _, r := range refs {
		if _, err := b.store.Delete(r); err != nil {
			failures = multierror.Append(failures, err)
		}
	}
	if failures != nil {
		return failures.ErrorOrNil()
	}

	path := b.storePath(id.Digest())
	_, err := os.Stat(b.storePath(id.Digest()))
	if os.IsNotExist(err) {
		return unknownReference(ref.String())
	}
	return os.RemoveAll(path)
}

func (b *imageStore) LookUp(refOrID string) (reference.Reference, error) {
	id, err := FromString(refOrID)
	if err == nil {
		_, err := os.Stat(b.storePath(id.Digest()))
		if os.IsNotExist(err) {
			return nil, unknownReference(refOrID)
		}
		return id, err
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
		if err == refstore.ErrDoesNotExist {
			return nil, unknownReference(refOrID)
		}
		return nil, err
	}
	return named, nil
}

func (b *imageStore) matchShortID(shortID string) (reference.Reference, error) {
	var found string
	ids, err := b.listIDs()
	if err != nil {
		return nil, err
	}
	for _, id := range ids {
		if strings.HasPrefix(id, shortID) {
			if found != "" && found != id {
				return nil, fmt.Errorf("ambiguous reference found")
			}
			found = id
		}
	}
	if found == "" {
		return nil, unknownReference(shortID)
	}
	ref := fromID(found)
	return ref, nil
}

func (b *imageStore) referenceToID(ref reference.Reference) (ID, error) {
	if id, ok := ref.(ID); ok {
		return id, nil
	}
	named := ref.(reference.Named)
	digest, err := b.store.Get(reference.TagNameOnly(named))
	return ID(digest), err
}

func (b *imageStore) storePath(ref digest.Digest) string {
	return filepath.Join(b.path, "contents", ref.Algorithm().String(), ref.Encoded())
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
