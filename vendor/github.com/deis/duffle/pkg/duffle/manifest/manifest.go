package manifest

import (
	"os"
	"path/filepath"

	"github.com/deis/duffle/pkg/bundle"

	"github.com/technosophos/moniker"
)

// Manifest represents a duffle manifest.
type Manifest struct {
	Name             string                                `json:"name" mapstructure:"name"`
	Version          string                                `json:"version" mapstructure:"version"`
	Description      string                                `json:"description,omitempty" mapstructure:"description,omitempty"`
	Keywords         []string                              `json:"keywords,omitempty" mapstructure:"keywords,omitempty"`
	Maintainers      []bundle.Maintainer                   `json:"maintainers,omitempty" mapstructure:"maintainers,omitempty"`
	InvocationImages map[string]*InvocationImage           `json:"invocationImages,omitempty" mapstructure:"invocationImages,omitempty"`
	Images           map[string]bundle.Image               `json:"images,omitempty" mapstructure:"images,omitempty"`
	Actions          map[string]bundle.Action              `json:"actions,omitempty" mapstructure:"actions,omitempty"`
	Parameters       map[string]bundle.ParameterDefinition `json:"parameters,omitempty" mapstructure:"parameters,omitempty"`
	Credentials      map[string]bundle.Location            `json:"credentials,omitempty" mapstructure:"credentials,omitempty"`
}

// Component represents a component of a CNAB bundle
type Component struct {
	Name          string            `mapstructure:"name"`
	Builder       string            `mapstructure:"builder"`
	Configuration map[string]string `mapstructure:"configuration"`
}

// New creates a new manifest with the Environments intialized.
func New() *Manifest {
	return &Manifest{
		Name: generateName(),
	}
}

// generateName generates a name based on the current working directory or a random name.
func generateName() string {
	var name string
	cwd, err := os.Getwd()
	if err == nil {
		name = filepath.Base(cwd)
	} else {
		namer := moniker.New()
		name = namer.NameSep("-")
	}
	return name
}
