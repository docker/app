package secrets

import (
	"crypto/sha256"
	"fmt"
)

// Secrets represents a secret map
type Secrets map[string]Secret

// Secret represents a secret
type Secret struct {
	Path     string
	External bool
	Name     string
}

// New creates a new empty secret map
func New() Secrets {
	return make(map[string]Secret)
}

// NormalizeFilename generates a filename for this secret to be mounted in the invocation image
func (s *Secret) NormalizeFilename() string {
	digest := sha256.Sum256([]byte(s.Path))
	return fmt.Sprintf("/cnab/app/secret/%x", digest)
}
