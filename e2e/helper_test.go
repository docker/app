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
	image       string
	privatePort int
	address     string
	container   string
}

// NewContainer creates a new Container
func NewContainer(image string, privatePort int) *Container {
	return &Container{
		image:       image,
		privatePort: privatePort,
	}
}

// Start starts a new docker container on a random port
func (c *Container) Start(t *testing.T) {
	result := icmd.RunCommand(dockerCli.path, "run", "--rm", "--privileged", "-d", "-P", c.image).Assert(t, icmd.Success)
	c.container = strings.Trim(result.Stdout(), " \r\n")
	time.Sleep(time.Second * 3)
}

// Stop terminates this container
func (c *Container) Stop(t *testing.T) {
	icmd.RunCommand(dockerCli.path, "stop", c.container).Assert(t, icmd.Success)
}

// GetAddress returns the host:port this container listens on
func (c *Container) GetAddress(t *testing.T) string {
	if c.address != "" {
		return c.address
	}
	result := icmd.RunCommand(dockerCli.path, "port", c.container, strconv.Itoa(c.privatePort)).Assert(t, icmd.Success)
	c.address = fmt.Sprintf("127.0.0.1:%v", strings.Trim(strings.Split(result.Stdout(), ":")[1], " \r\n"))
	return c.address
}
