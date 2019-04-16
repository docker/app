package store

import (
	_ "crypto/sha256" // ensure ids can be computed
	"os"
	"path/filepath"

	"github.com/deislabs/duffle/pkg/claim"
	"github.com/deislabs/duffle/pkg/utils/crud"
	digest "github.com/opencontainers/go-digest"
	"github.com/pkg/errors"
)

const (
	// AppConfigDirectory is the Docker App directory name inside Docker config directory
	AppConfigDirectory = "app"
	// BundleStoreDirectory is the bundle store directory name
	BundleStoreDirectory = "bundles"
	// CredentialStoreDirectory is the credential store directory name
	CredentialStoreDirectory = "credentials"
	// InstallationStoreDirectory is the installations store directory name
	InstallationStoreDirectory = "installations"
)

// ApplicationStore is the main point to access different stores:
// - Bundle store persists all bundles built or fetched locally
// - Credential store persists all the credentials, per context basis
// - Installation store persists all the installations, per context basis
type ApplicationStore struct {
	path string
}

// NewApplicationStore creates a new application store, nested inside a
// docker configuration directory. It will create all the directory hierarchy
// if anything is missing.
func NewApplicationStore(configDir string) (*ApplicationStore, error) {
	storePath := filepath.Join(configDir, AppConfigDirectory)
	directories := []struct {
		dir  string
		perm os.FileMode
	}{
		{BundleStoreDirectory, 0755},
		{CredentialStoreDirectory, 0700},
		{InstallationStoreDirectory, 0755},
	}
	for _, d := range directories {
		if err := os.MkdirAll(filepath.Join(storePath, d.dir), d.perm); err != nil {
			return nil, errors.Wrapf(err, "failed to create application store directory %q", d.dir)
		}
	}
	return &ApplicationStore{path: storePath}, nil
}

// InstallationStore initializes and returns a context based installation store
func (a ApplicationStore) InstallationStore(context string) (InstallationStore, error) {
	path := filepath.Join(a.path, InstallationStoreDirectory, makeDigestedDirectory(context))
	if err := os.MkdirAll(path, 0755); err != nil {
		return nil, errors.Wrapf(err, "failed to create installation store directory for context %q", context)
	}
	return claim.NewClaimStore(crud.NewFileSystemStore(path, "json")), nil
}

// CredentialStore initializes and returns a context based credential store
func (a ApplicationStore) CredentialStore(context string) (CredentialStore, error) {
	path := filepath.Join(a.path, CredentialStoreDirectory, makeDigestedDirectory(context))
	if err := os.MkdirAll(path, 0700); err != nil {
		return nil, errors.Wrapf(err, "failed to create credential store directory for context %q", context)
	}
	return &credentialStore{path: path}, nil
}

// BundleStore initializes and returns a bundle store
func (a ApplicationStore) BundleStore() (BundleStore, error) {
	path := filepath.Join(a.path, BundleStoreDirectory)
	if err := os.MkdirAll(path, 0755); err != nil {
		return nil, errors.Wrapf(err, "failed to create bundle store directory %q", path)
	}
	return &bundleStore{path: path}, nil
}

func makeDigestedDirectory(context string) string {
	return digest.FromString(context).Encoded()
}
