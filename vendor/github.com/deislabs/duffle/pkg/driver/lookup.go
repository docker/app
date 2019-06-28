package driver

import (
	"fmt"

	"github.com/deislabs/cnab-go/driver"
)

// Lookup takes a driver name and tries to resolve the most pertinent driver.
func Lookup(name string) (driver.Driver, error) {
	switch name {
	case "docker":
		return &DockerDriver{}, nil
	case "kubernetes", "k8s":
		return &KubernetesDriver{}, nil
	case "debug":
		return &driver.DebugDriver{}, nil
	case "command":
		return &CommandDriver{Name: name}, nil
	default:
		return nil, fmt.Errorf("unsupported driver: %s", name)
	}
}
