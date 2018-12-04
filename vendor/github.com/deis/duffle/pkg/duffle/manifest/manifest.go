package manifest

import (
	"os"
	"path/filepath"

	"github.com/deis/duffle/pkg/bundle"

	"github.com/technosophos/moniker"
)

// Manifest represents a duffle manifest.
type Manifest struct {
	Name        string                                `mapstructure:"name"`
	Version     string                                `mapstructure:"version"`
	Description string                                `mapstructure:"description"`
	Keywords    []string                              `mapstructure:"keywords"`
	Maintainers []bundle.Maintainer                   `mapstructure:"maintainers"`
	Components  map[string]*Component                 `mapstructure:"components"`
	Parameters  map[string]bundle.ParameterDefinition `mapstructure:"parameters"`
	Credentials map[string]bundle.Location            `mapstructure:"credentials"`
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
