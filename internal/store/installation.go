package store

import (
	"fmt"

	"github.com/deislabs/duffle/pkg/claim"
)

// InstallationStore is an interface to persist claims.
type InstallationStore interface {
	List() ([]string, error)
	Store(installation claim.Claim) error
	Read(installationName string) (claim.Claim, error)
	Delete(installationName string) error
}

var _ InstallationStore = &installationStore{}

type installationStore struct {
	claimStore claim.Store
}

func (i installationStore) List() ([]string, error) {
	return i.claimStore.List()
}

func (i installationStore) Store(installation claim.Claim) error {
	return i.claimStore.Store(installation)
}

func (i installationStore) Read(installationName string) (claim.Claim, error) {
	c, err := i.claimStore.Read(installationName)
	if err == claim.ErrClaimNotFound {
		return claim.Claim{}, fmt.Errorf("Installation %q not found", installationName)
	}
	return c, err
}

func (i installationStore) Delete(installationName string) error {
	return i.claimStore.Delete(installationName)
}
