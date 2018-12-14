package loader

import (
	"github.com/deislabs/duffle/pkg/bundle"
	"github.com/deislabs/duffle/pkg/signature"
)

// SecureLoader loads signed bundles.
//
// A signed bundle is a bundle.json file that has been cryptographically signed.
// This loader will load such a file, and test the validity of the signature.
type SecureLoader struct {
	keyring *signature.KeyRing
}

// NewSecureLoader creates a new SecureLoader.NewSecureLoader.
// A SecureLoader uses a keyring to validate the signature on a signed bundle.
func NewSecureLoader(keyring *signature.KeyRing) *SecureLoader {
	return &SecureLoader{
		keyring: keyring,
	}
}

// Load will load a file, verify the signature, and return the parsed *Bundle.
func (s *SecureLoader) Load(filename string) (*bundle.Bundle, error) {
	b := &bundle.Bundle{}
	data, err := loadData(filename)
	if err != nil {
		return b, err
	}
	return s.LoadData(data)
}

// LoadData loads a bundle from data.
//
// This will perform verification of the bundle, extract the JSON, and then
// parse the JSON into a *Bundle.
func (s *SecureLoader) LoadData(data []byte) (*bundle.Bundle, error) {
	verifier := signature.NewVerifier(s.keyring)
	b, _, err := verifier.Extract(data)
	return b, err
}
