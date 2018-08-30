package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

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

	binName := FindBinary("docker-app", os.Getenv("DOCKERAPP_BINARY"))
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

func startRegistry(t *testing.T) *Container {
	c := &Container{image: "registry:2", privatePort: 5000}
	c.Start(t)
	return c
}
