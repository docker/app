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

	"github.com/deislabs/cnab-go/credentials"
	"github.com/docker/app/internal/store"
	dockerConfigFile "github.com/docker/cli/cli/config/configfile"
	"gotest.tools/assert"
	"gotest.tools/icmd"
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

type ConfigFileOperator func(configFile *dockerConfigFile.ConfigFile)

func (d dockerCliCommand) createTestCmd(ops ...ConfigFileOperator) (icmd.Cmd, func()) {
	configDir, err := ioutil.TempDir("", "config")
	if err != nil {
		panic(err)
	}
	configFilePath := filepath.Join(configDir, "config.json")
	config := dockerConfigFile.ConfigFile{
		CLIPluginsExtraDirs: []string{
			d.cliPluginDir,
		},
		Filename: configFilePath,
	}
	for _, op := range ops {
		op(&config)
	}
	configFile, err := os.Create(configFilePath)
	if err != nil {
		panic(err)
	}
	defer configFile.Close()
	err = json.NewEncoder(configFile).Encode(config)
	if err != nil {
		panic(err)
	}
	cleanup := func() {
		os.RemoveAll(configDir)
	}
	env := append(os.Environ(),
		"DOCKER_CONFIG="+configDir,
		"DOCKER_CLI_EXPERIMENTAL=enabled") // TODO: Remove this once docker app plugin is no more experimental
	return icmd.Cmd{Env: env}, cleanup
}

func (d dockerCliCommand) Command(args ...string) []string {
	return append([]string{d.path}, args...)
}

func withCredentialSet(t *testing.T, context string, creds *credentials.CredentialSet) ConfigFileOperator {
	t.Helper()
	return func(config *dockerConfigFile.ConfigFile) {
		configDir := filepath.Dir(config.Filename)
		appstore, err := store.NewApplicationStore(configDir)
		assert.NilError(t, err)

		credstore, err := appstore.CredentialStore(context)
		assert.NilError(t, err)

		err = credstore.Store(creds)
		assert.NilError(t, err)
	}
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
