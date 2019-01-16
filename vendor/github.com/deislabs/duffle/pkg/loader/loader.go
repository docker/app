package loader

import (
	"github.com/deislabs/duffle/pkg/bundle"
	"github.com/deislabs/duffle/pkg/signature"
)

// Loader provides an interface for loading a bundle
type Loader interface {
	// Load a bundle from a local file
	Load(source string) (*bundle.Bundle, error)
	// Load a bundle from raw data
	LoadData(data []byte) (*bundle.Bundle, error)
}

// New creates a loader for signed bundle files.
func New(keyring *signature.KeyRing) Loader {
	return &SecureLoader{
		keyring: keyring,
	}
}
