package e2e

import (
	"bytes"
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var (
	e2ePath         = flag.String("e2e-path", ".", "Set path to the e2e directory")
	dockerApp       = os.Getenv("DOCKERAPP_BINARY")
	dockerCli       = os.Getenv("DOCKERCLI_BINARY")
	hasExperimental = false
	renderers       = ""
)

func TestMain(m *testing.M) {
	flag.Parse()
	if err := os.Chdir(*e2ePath); err != nil {
		panic(err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	if dockerApp == "" {
		dockerApp = filepath.Join(cwd, "../bin/docker-app")
	}
	dockerApp, err = filepath.Abs(dockerApp)
	if err != nil {
		panic(err)
	}
	if dockerCli == "" {
		dockerCli = "docker"
	}
	cmd := exec.Command(dockerApp, "version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		panic(err)
	}
	hasExperimental = bytes.Contains(output, []byte("Experimental: on"))
	i := strings.Index(string(output), "Renderers")
	renderers = string(output)[i+10:]
	os.Exit(m.Run())
}
