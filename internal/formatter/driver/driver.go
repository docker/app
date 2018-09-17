package driver

import (
	composetypes "github.com/docker/cli/cli/compose/types"
)

// Driver is the interface that must be implemented by a formatter driver.
type Driver interface {
	// Format executes the formatter on the source config
	Format(config *composetypes.Config) (string, error)
}
