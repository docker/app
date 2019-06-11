package store

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"

	"github.com/deislabs/cnab-go/credentials"
	"github.com/pkg/errors"
)

// CredentialStore persists credential sets to a specific path.
type CredentialStore interface {
	Store(creds *credentials.CredentialSet) error
	Read(credentialSetName string) (*credentials.CredentialSet, error)
}

var _ CredentialStore = &credentialStore{}

type credentialStore struct {
	path string
}

func (c *credentialStore) Read(credentialSetName string) (*credentials.CredentialSet, error) {
	path := filepath.Join(c.path, credentialSetName+".yaml")
	return credentials.Load(path)
}

func (c *credentialStore) Store(creds *credentials.CredentialSet) error {
	if creds.Name == "" {
		return errors.New("failed to store credential set, name is empty")
	}
	data, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return errors.Wrapf(err, "failed to store credential set %q", creds.Name)
	}
	err = ioutil.WriteFile(filepath.Join(c.path, creds.Name+".yaml"), data, 0644)
	return errors.Wrapf(err, "failed to store credential set %q", creds.Name)
}
