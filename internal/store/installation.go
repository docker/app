package store

import (
	"encoding/json"
	"fmt"

	"github.com/deislabs/duffle/pkg/claim"
	"github.com/deislabs/duffle/pkg/utils/crud"
)

// InstallationStore is an interface to persist, delete, list and read installations.
type InstallationStore interface {
	List() ([]string, error)
	Store(installation *Installation) error
	Read(installationName string) (*Installation, error)
	Delete(installationName string) error
}

// Installation is a CNAB claim with an information of where the bundle comes from.
// It persists the result of an installation and its parameters and context.
type Installation struct {
	claim.Claim
	Reference string `json:"reference,omitempty"`
}

func NewInstallation(name string, reference string) (*Installation, error) {
	c, err := claim.New(name)
	if err != nil {
		return nil, err
	}
	return &Installation{
		Claim:     *c,
		Reference: reference,
	}, nil
}

var _ InstallationStore = &installationStore{}

type installationStore struct {
	store crud.Store
}

func (i installationStore) List() ([]string, error) {
	return i.store.List()
}

func (i installationStore) Store(installation *Installation) error {
	data, err := json.MarshalIndent(installation, "", "  ")
	if err != nil {
		return err
	}
	return i.store.Store(installation.Name, data)
}

func (i installationStore) Read(installationName string) (*Installation, error) {
	data, err := i.store.Read(installationName)
	if err != nil {
		if err == crud.ErrFileDoesNotExist {
			return nil, fmt.Errorf("Installation %q not found", installationName)

		}
		return nil, err
	}
	var installation Installation
	if err := json.Unmarshal(data, &installation); err != nil {
		return nil, err
	}
	return &installation, nil
}

func (i installationStore) Delete(installationName string) error {
	return i.store.Delete(installationName)
}
