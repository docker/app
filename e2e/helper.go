package e2e

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"gotest.tools/assert"
	"gotest.tools/golden"
)

func binExt() string {
	if runtime.GOOS == "windows" {
		return ".exe"
	}
	return ""
}

func runCmd(t *testing.T, cmd *exec.Cmd) string {
	var outputBuf, errBuf bytes.Buffer
	cmd.Stdout = &outputBuf
	cmd.Stderr = &errBuf
	err := cmd.Run()
	assert.NilError(t, err, errBuf.String())
	return outputBuf.String()
}

// FindBinary looks for given binary trying variations on name and location
func FindBinary(app string, options ...string) string {
	binNames := append(options, []string{
		fmt.Sprintf("../%s-%s%s", app, runtime.GOOS, binExt()),
		fmt.Sprintf("../%s%s", app, binExt()),
		fmt.Sprintf("../bin/%s-%s%s", app, runtime.GOOS, binExt()),
		fmt.Sprintf("../bin/%s%s", app, binExt()),
	}...)
	for _, binName := range binNames {
		if s, err := os.Stat(binName); err == nil && !s.IsDir() {
			return binName
		}
	}
	return ""
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
	cmd := exec.Command("docker", "run", "--rm", "--privileged", "-d", "-P", c.image)
	output := runCmd(t, cmd)
	c.container = strings.Trim(output, " \r\n")
	time.Sleep(time.Second * 3)
}

// Stop terminates this container
func (c *Container) Stop(t *testing.T) {
	runCmd(t, exec.Command("docker", "stop", c.container))
}

// GetAddress returns the host:port this container listens on
func (c *Container) GetAddress(t *testing.T) string {
	if c.address != "" {
		return c.address
	}
	cmd := exec.Command("docker", "port", c.container, strconv.Itoa(c.privatePort))
	output := runCmd(t, cmd)
	c.address = fmt.Sprintf("127.0.0.1:%v", strings.Trim(strings.Split(output, ":")[1], " \r\n"))
	return c.address
}

// AssertCommand runs command, assert it succeeds, return its output
func AssertCommand(t *testing.T, exe string, args ...string) []byte {
	t.Helper()
	cmd := exec.Command(exe, args...)
	output, err := cmd.CombinedOutput()
	assert.NilError(t, err, string(output))
	return output
}

// AssertCommandOutput runs commands and asserts its output match expectations
func AssertCommandOutput(t *testing.T, goldenFile string, cmd string, args ...string) {
	t.Helper()
	output := AssertCommand(t, cmd, args...)
	golden.Assert(t, string(output), goldenFile)
}

// AssertCommandFailureOutput runs commands and asserts it fails with given output
func AssertCommandFailureOutput(t *testing.T, goldenFile string, exe string, args ...string) {
	t.Helper()
	cmd := exec.Command(exe, args...)
	output, err := cmd.CombinedOutput()
	assert.Assert(t, err != nil)
	golden.Assert(t, string(output), goldenFile)
}
