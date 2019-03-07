package e2e

import (
	"bytes"
	"encoding/json"
	"flag"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	dockerConfigFile "github.com/docker/cli/cli/config/configfile"
)

var (
	e2ePath         = flag.String("e2e-path", ".", "Set path to the e2e directory")
	dockerCliPath   = os.Getenv("DOCKERCLI_BINARY")
	hasExperimental = false
	renderers       = ""
	dockerCli       dockerCliCommand
)

type dockerCliCommand struct {
	path         string
	cliPluginDir string
}

func (d dockerCliCommand) createTestConfig() string {
	configDir, err := ioutil.TempDir("", "config")
	if err != nil {
		panic(err)
	}
	config := dockerConfigFile.ConfigFile{CLIPluginsExtraDirs: []string{d.cliPluginDir}}
	configFile, err := os.Create(filepath.Join(configDir, "config.json"))
	if err != nil {
		panic(err)
	}
	err = json.NewEncoder(configFile).Encode(config)
	if err != nil {
		panic(err)
	}
	err = os.Setenv("DOCKER_CONFIG", configDir)
	if err != nil {
		panic(err)
	}
	return configDir
}

func (d dockerCliCommand) Command(args ...string) []string {
	return append([]string{d.path}, args...)
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
	if dockerCliPath == "" {
		dockerCliPath = "docker"
	} else {
		dockerCliPath, err = filepath.Abs(dockerCliPath)
		if err != nil {
			panic(err)
		}
	}
	// Prepare docker cli to call the docker-app plugin binary:
	// - Create a symbolic link with the dockerApp binary to the plugin directory
	cliPluginDir, err := ioutil.TempDir("", "configContent")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(cliPluginDir)
	createDockerAppSymLink(dockerApp, cliPluginDir)

	dockerCli = dockerCliCommand{path: dockerCliPath, cliPluginDir: cliPluginDir}

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

func createDockerAppSymLink(dockerApp, configDir string) {
	dockerAppExecName := "docker-app"
	if runtime.GOOS == "windows" {
		dockerAppExecName += ".exe"
	}
	if err := os.Symlink(dockerApp, filepath.Join(configDir, dockerAppExecName)); err != nil {
		panic(err)
	}
}
