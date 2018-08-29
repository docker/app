package e2e

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"gotest.tools/assert"
)

var (
	dockerApp       = ""
	hasExperimental = false
	renderers       = ""
)

func getDockerAppBinary(t *testing.T) (string, bool) {
	t.Helper()
	if dockerApp != "" {
		return dockerApp, hasExperimental
	}

	binName := findBinary("docker-app", os.Getenv("DOCKERAPP_BINARY"))
	assert.Assert(t, binName != "", "cannot locate docker-app binary")
	var err error
	binName, err = filepath.Abs(binName)
	assert.NilError(t, err, "failed to convert dockerApp path to absolute")
	cmd := exec.Command(binName, "version")
	output, err := cmd.CombinedOutput()
	assert.NilError(t, err, "failed to execute %s", binName)
	dockerApp = binName
	sOutput := string(output)
	hasExperimental = strings.Contains(sOutput, "Experimental: on")
	i := strings.Index(sOutput, "Renderers")
	renderers = sOutput[i+10:]
	return dockerApp, hasExperimental
}

func findBinary(app string, options ...string) string {
	binNames := append(options, []string{
		fmt.Sprintf("../%s-%s%s", app, runtime.GOOS, binExt()),
		fmt.Sprintf("../%s%s", app, binExt()),
		fmt.Sprintf("../bin/%s-%s%s", app, runtime.GOOS, binExt()),
		fmt.Sprintf("../bin/%s%s", app, binExt()),
	}...)
	for _, binName := range binNames {
		if _, err := os.Stat(binName); err == nil {
			return binName
		}
	}
	return ""
}

func binExt() string {
	if runtime.GOOS == "windows" {
		return ".exe"
	}
	return ""
}

type container struct {
	image       string
	privatePort int
	address     string
	container   string
}

func startRegistry(t *testing.T) *container {
	c := &container{image: "registry:2", privatePort: 5000}
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
