package metadata

import (
	"strings"

	"github.com/cnabio/cnab-go/bundle"
)

// Maintainer represents one of the apps's maintainers
type Maintainer struct {
	Name  string
	Email string `json:",omitempty"`
}

// Maintainers is a list of maintainers
type Maintainers []Maintainer

// String gives a string representation of a list of maintainers
func (ms Maintainers) String() string {
	res := make([]string, len(ms))
	for i, m := range ms {
		res[i] = m.String()
	}
	return strings.Join(res, ", ")
}

// String gives a string representation of a maintainer
func (m Maintainer) String() string {
	s := m.Name
	if m.Email != "" {
		s += " <" + m.Email + ">"
	}
	return s
}

// AppMetadata is the format of the data found inside the metadata.yml file
type AppMetadata struct {
	Version     string
	Name        string
	Description string      `json:",omitempty"`
	Maintainers Maintainers `json:",omitempty"`
}

// FromBundle extracts the docker-app metadata from the bundle
func FromBundle(bndl *bundle.Bundle) AppMetadata {
	meta := AppMetadata{
		Name:        bndl.Name,
		Version:     bndl.Version,
		Description: bndl.Description,
	}
	for _, m := range bndl.Maintainers {
		meta.Maintainers = append(meta.Maintainers, Maintainer{
			Name:  m.Name,
			Email: m.Email,
		})
	}
	return meta
}
