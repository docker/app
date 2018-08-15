package metadata

import (
	"strings"
)

// Maintainer represents one of the apps's maintainers
type Maintainer struct {
	Name  string
	Email string
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
	Description string
	Namespace   string
	Maintainers Maintainers
	Parents     Parents
}

// Parents is a list of ParentMetadata items
type Parents []ParentMetadata

// ParentMetadata contains historical data of forked packages
type ParentMetadata struct {
	Name        string
	Namespace   string
	Version     string
	Maintainers Maintainers
}

// Modifier is a function signature that takes and returns an AppMetadata object
type Modifier func(AppMetadata) AppMetadata

// From returns an AppMetadata instance based on the provided AppMetadata
// and applicable modifier functions
func From(orig AppMetadata, modifiers ...Modifier) AppMetadata {
	parent := ParentMetadata{
		Name:        orig.Name,
		Namespace:   orig.Namespace,
		Version:     orig.Version,
		Maintainers: orig.Maintainers,
	}

	result := AppMetadata{
		Version:     orig.Version,
		Name:        orig.Name,
		Namespace:   orig.Namespace,
		Description: orig.Description,
		Maintainers: orig.Maintainers,
		Parents:     append(orig.Parents, parent),
	}
	for _, f := range modifiers {
		result = f(result)
	}
	return result
}

// WithMaintainers returns a modified AppMetadata with updated maintainers field
func WithMaintainers(maintainers Maintainers) Modifier {
	return func(parent AppMetadata) AppMetadata {
		parent.Maintainers = maintainers
		return parent
	}
}

// WithName returns a modified AppMetadata with updated name field
func WithName(name string) Modifier {
	return func(parent AppMetadata) AppMetadata {
		parent.Name = name
		return parent
	}
}

// WithNamespace returns a modified AppMetadata with updated namespace field
func WithNamespace(namespace string) Modifier {
	return func(parent AppMetadata) AppMetadata {
		parent.Namespace = namespace
		return parent
	}
}
