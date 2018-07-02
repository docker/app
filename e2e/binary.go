package e2e

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"gotest.tools/assert"
)

var (
	dockerApp       = ""
	yamlschemaApp   = ""
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

func getYamlschemaBinary(t *testing.T) string {
	t.Helper()
	if yamlschemaApp != "" {
		return yamlschemaApp
	}
	binName := findBinary("yamlschema")
	assert.Assert(t, binName != "", "cannot locate yamlschema binary")
	var err error
	binName, err = filepath.Abs(binName)
	assert.NilError(t, err, "failed to convert yamlschema path to absolute")
	yamlschemaApp = binName
	return yamlschemaApp
}

func findBinary(app string, options ...string) string {
	binNames := append(options, []string{
		fmt.Sprintf("./%s-%s%s", app, runtime.GOOS, binExt()),
		fmt.Sprintf("./%s%s", app, binExt()),
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
