package store

import "github.com/deislabs/duffle/pkg/claim"

// InstallationStore is an interface to persist claims.
type InstallationStore interface {
	List() ([]string, error)
	Store(installationName claim.Claim) error
	Read(installationName string) (claim.Claim, error)
	Delete(installationName string) error
}

var _ InstallationStore = &claim.Store{}
