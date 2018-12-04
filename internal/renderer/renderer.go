package renderer

import (
	"sort"
	"sync"

	"github.com/docker/app/internal/renderer/driver"
	"github.com/pkg/errors"
)

var (
	driversMu sync.RWMutex
	drivers   = map[string]driver.Driver{}
)

// Register makes a renderer available by the provided name.
// If Register is called twice with the same name or if driver is nil,
// it panics.
func Register(name string, driver driver.Driver) {
	driversMu.Lock()
	defer driversMu.Unlock()
	if driver == nil {
		panic("renderer: Register driver is nil")
	}
	if _, dup := drivers[name]; dup {
		panic("renderer: Register called twice for driver " + name)
	}
	drivers[name] = driver
}

// Apply applies the specified render to the specified string with the specified parameters.
// If the render is not present is the registered ones, it errors out.
func Apply(s string, parameters map[string]interface{}, renderers ...string) (string, error) {
	var err error
	for _, r := range renderers {
		if r == "none" {
			continue
		}
		driversMu.RLock()
		d, present := drivers[r]
		driversMu.RUnlock()
		if !present {
			return "", errors.Errorf("unknown renderer %s", r)
		}
		s, err = d.Apply(s, parameters)
		if err != nil {
			return "", err
		}
	}
	return s, nil
}

// Drivers returns a sorted list of the names of the registered drivers.
func Drivers() []string {
	list := []string{"none"}
	driversMu.RLock()
	for name := range drivers {
		list = append(list, name)
	}
	driversMu.RUnlock()
	sort.Strings(list)
	return list
}
