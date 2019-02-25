package e2e

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var (
	e2ePath         = flag.String("e2e-path", ".", "Set path to the e2e directory")
	dockerCliPath   = os.Getenv("DOCKERCLI_BINARY")
	hasExperimental = false
	renderers       = ""
	dockerCli       dockerCliCommand
)

const config = `{
		"cliPluginsExtraDirs": ["%s"]
}`

type dockerCliCommand struct {
	path   string
	config string
}

func (d dockerCliCommand) Command(args ...string) []string {
	return append([]string{d.path, "--config", d.config}, args...)
}

func TestMain(m *testing.M) {
	flag.Parse()
	if err := os.Chdir(*e2ePath); err != nil {
		panic(err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	dockerApp := os.Getenv("DOCKERAPP_BINARY")
	if dockerApp == "" {
		dockerApp = filepath.Join(cwd, "../bin/docker-app")
	}
	dockerApp, err = filepath.Abs(dockerApp)
	if err != nil {
		panic(err)
	}
	// Prepare docker cli to call the docker-app plugin binary:
	// - Create a config dir with a custom config file
	// - Create a symbolic link with the dockerApp binary to the plugin directory
	if dockerCliPath == "" {
		dockerCliPath = "docker"
	}
	configDir, err := ioutil.TempDir("", "config")
	if err != nil {
		panic(err.Error())
	}
	defer os.RemoveAll(configDir)
	dockerCli = dockerCliCommand{path: dockerCliPath, config: configDir}
	ioutil.WriteFile(filepath.Join(configDir, "config.json"), []byte(fmt.Sprintf(config, configDir)), 0644)
	if err := os.Symlink(dockerApp, filepath.Join(configDir, "docker-app")); err != nil {
		panic(err.Error())
	}

	cmd := exec.Command(dockerApp, "app", "version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		panic(err)
	}
	hasExperimental = bytes.Contains(output, []byte("Experimental: on"))
	i := strings.Index(string(output), "Renderers")
	renderers = string(output)[i+10:]
	os.Exit(m.Run())
}
