package e2e

import (
	"bytes"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	"gotest.tools/assert"
)

type container struct {
	image       string
	privatePort int
	address     string
	container   string
}

//nolint: deadcode
func startRegistry(t *testing.T) *container {
	c := &container{image: "registry:2", privatePort: 5000}
	c.start(t)
	return c
}

//nolint: deadcode
func startDind(t *testing.T) *container {
	c := &container{image: "docker:dind", privatePort: 2375}
	c.start(t)
	return c
}

// Start starts a new docker container on a random port
func (c *container) start(t *testing.T) {
	cmd := exec.Command("docker", "run", "--rm", "--privileged", "-d", "-P", c.image)
	output := runCmd(t, cmd)
	c.container = strings.Trim(output, " \r\n")
	time.Sleep(time.Second * 3)
}

// Stop terminates this container
func (c *container) stop(t *testing.T) {
	runCmd(t, exec.Command("docker", "stop", c.container))
}

// getAddress returns the host:port this container listens on
func (c *container) getAddress(t *testing.T) string {
	if c.address != "" {
		return c.address
	}
	cmd := exec.Command("docker", "port", c.container, strconv.Itoa(c.privatePort))
	output := runCmd(t, cmd)
	c.address = fmt.Sprintf("127.0.0.1:%v", strings.Trim(strings.Split(output, ":")[1], " \r\n"))
	return c.address
}

func runCmd(t *testing.T, cmd *exec.Cmd) string {
	var outputBuf, errBuf bytes.Buffer
	cmd.Stdout = &outputBuf
	cmd.Stderr = &errBuf
	err := cmd.Run()
	assert.NilError(t, err, errBuf.String())
	return outputBuf.String()
}
