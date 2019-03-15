package e2e

import (
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
	"testing"
	"time"

	"gotest.tools/assert"
	"gotest.tools/icmd"
)

// readFile returns the content of the file at the designated path normalizing
// line endings by removing any \r.
func readFile(t *testing.T, path string) string {
	t.Helper()
	content, err := ioutil.ReadFile(path)
	assert.NilError(t, err, "missing '"+path+"' file")
	return strings.Replace(string(content), "\r", "", -1)
}

// checkRenderers returns false if appname requires a renderer that is not in enabled
func checkRenderers(appname string, enabled string) bool {
	renderers := []string{"gotemplate", "yatee", "mustache"}
	for _, r := range renderers {
		if strings.Contains(appname, r) && !strings.Contains(enabled, r) {
			return false
		}
	}
	return true
}

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
	result := icmd.RunCommand(dockerCli.path, args...).Assert(t, icmd.Success)
	c.container = strings.Trim(result.Stdout(), " \r\n")
	time.Sleep(time.Second * 3)
}

// StartWithContainerNetwork starts a new container using an existing container network
func (c *Container) StartWithContainerNetwork(t *testing.T, other *Container, dockerArgs ...string) {
	args := []string{"run", "--rm", "--privileged", "-d", "--network=container:" + other.container}
	args = append(args, dockerArgs...)
	args = append(args, c.image)
	args = append(args, c.args...)
	result := icmd.RunCommand(dockerCli.path, args...).Assert(t, icmd.Success)
	c.container = strings.Trim(result.Stdout(), " \r\n")
	time.Sleep(time.Second * 3)
	c.parentContainer = other.container
}

// Stop terminates this container
func (c *Container) Stop(t *testing.T) {
	icmd.RunCommand(dockerCli.path, "stop", c.container).Assert(t, icmd.Success)
}

// StopNoFail terminates this container
func (c *Container) StopNoFail() {
	icmd.RunCommand(dockerCli.path, "stop", c.container)
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
	result := icmd.RunCommand(dockerCli.path, "port", container, strconv.Itoa(c.privatePort)).Assert(t, icmd.Success)
	c.address = fmt.Sprintf("127.0.0.1:%v", strings.Trim(strings.Split(result.Stdout(), ":")[1], " \r\n"))
	return c.address
}

// GetPrivateAddress returns the host:port this container listens on
func (c *Container) GetPrivateAddress(t *testing.T) string {
	container := c.parentContainer
	if container == "" {
		container = c.container
	}
	result := icmd.RunCommand(dockerCli.path, "inspect", container, "-f", "{{.NetworkSettings.IPAddress}}").Assert(t, icmd.Success)
	return fmt.Sprintf("%s:%d", strings.TrimSpace(result.Stdout()), c.privatePort)
}
