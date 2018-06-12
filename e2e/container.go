package e2e

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"testing"

	"gotest.tools/assert"
)

type container struct {
	privatePort int
	address     string
	container   string
}

func startRegistry(t *testing.T) *container {
	c := &container{privatePort: 5000}
	c.Start(t, "registry:2")
	return c
}

func startDind(t *testing.T) *container {
	c := &container{privatePort: 2375}
	c.Start(t, "docker:dind")
	return c
}

// Start starts a new docker container on a random port
func (c *container) Start(t *testing.T, image string) {
	cmd := exec.Command("docker", "run", "--rm", "--privileged", "-d", "-P", image)
	output, err := cmd.CombinedOutput()
	assert.NilError(t, err, string(output))
	c.container = strings.Trim(string(output), " \r\n")
}

// Stop terminates this container
func (c *container) Stop(t *testing.T) {
	cmd := exec.Command("docker", "stop", c.container)
	output, err := cmd.CombinedOutput()
	assert.NilError(t, err, string(output))
}

// Address returns the host:port this container listens on
func (c *container) Address(t *testing.T) string {
	if c.address != "" {
		return c.address
	}
	cmd := exec.Command("docker", "port", c.container, strconv.Itoa(c.privatePort))
	output, err := cmd.CombinedOutput()
	assert.NilError(t, err, string(output))
	c.address = fmt.Sprintf("localhost:%v", strings.Trim(strings.Split(string(output), ":")[1], " \r\n"))
	return c.address
}
