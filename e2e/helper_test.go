package e2e

import (
	"testing"
)

func startRegistry(t *testing.T) *Container {
	c := &Container{image: "registry:2", privatePort: 5000}
	c.Start(t)
	return c
}
