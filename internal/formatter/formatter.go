package formatter

import (
	"sort"
	"sync"

	"github.com/docker/app/internal/formatter/driver"
	composetypes "github.com/docker/cli/cli/compose/types"
	"github.com/pkg/errors"
)

var (
	driversMu sync.RWMutex
	drivers   = map[string]driver.Driver{}
)

// Register makes a formatter available by the provided name.
// If Register is called twice with the same name or if driver is nil,
// it panics.
func Register(name string, driver driver.Driver) {
	driversMu.Lock()
	defer driversMu.Unlock()
	if driver == nil {
		panic("formatter: Register driver is nil")
	}
	if _, dup := drivers[name]; dup {
		panic("formatter: Register called twice for driver " + name)
	}
	drivers[name] = driver
}

// Format uses the specified formatter to create a printable output.
// If the formatter is not registered, this errors out.
func Format(config *composetypes.Config, formatter string) (string, error) {
	driversMu.RLock()
	d, ok := drivers[formatter]
	driversMu.RUnlock()
	if !ok {
		return "", errors.Errorf("unknown formatter %q", formatter)
	}
	s, err := d.Format(config)
	if err != nil {
		return "", err
	}
	return s, nil
}

// Drivers returns a sorted list of the names of the registered drivers.
func Drivers() []string {
	list := []string{}
	driversMu.RLock()
	for name := range drivers {
		list = append(list, name)
	}
	driversMu.RUnlock()
	sort.Strings(list)
	return list
}
