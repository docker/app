package e2e

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"gotest.tools/icmd"
)

// Container represents a docker container
type Container struct {
	image           string
	privatePort     int
	address         string
	container       string
	parentContainer string
	args            []string
}

// NewContainer creates a new Container
func NewContainer(image string, privatePort int, args ...string) *Container {
	return &Container{
		image:       image,
		privatePort: privatePort,
		args:        args,
	}
}

// Start starts a new docker container on a random port
func (c *Container) Start(t *testing.T, dockerArgs ...string) {
	args := []string{"run", "--rm", "--privileged", "-d", "-P"}
	args = append(args, dockerArgs...)
	args = append(args, c.image)
	args = append(args, c.args...)
	result := icmd.RunCommand("docker", args...).Assert(t, icmd.Success)
	c.container = strings.Trim(result.Stdout(), " \r\n")
	time.Sleep(time.Second * 3)
}

// StartWithContainerNetwork starts a new container using an existing container network
func (c *Container) StartWithContainerNetwork(t *testing.T, other *Container, dockerArgs ...string) {
	args := []string{"run", "--rm", "--privileged", "-d", "--network=container:" + other.container}
	args = append(args, dockerArgs...)
	args = append(args, c.image)
	args = append(args, c.args...)
	result := icmd.RunCommand("docker", args...).Assert(t, icmd.Success)
	c.container = strings.Trim(result.Stdout(), " \r\n")
	time.Sleep(time.Second * 3)
	c.parentContainer = other.container
}

// Stop terminates this container
func (c *Container) Stop(t *testing.T) {
	icmd.RunCommand("docker", "stop", c.container).Assert(t, icmd.Success)
}

// GetAddress returns the host:port this container listens on
func (c *Container) GetAddress(t *testing.T) string {
	if c.address != "" {
		return c.address
	}
	container := c.parentContainer
	if container == "" {
		container = c.container
	}
	result := icmd.RunCommand("docker", "port", container, strconv.Itoa(c.privatePort)).Assert(t, icmd.Success)
	c.address = fmt.Sprintf("127.0.0.1:%v", strings.Trim(strings.Split(result.Stdout(), ":")[1], " \r\n"))
	return c.address
}
