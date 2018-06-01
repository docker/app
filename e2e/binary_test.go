package e2e

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/gotestyourself/gotestyourself/assert"
	"github.com/gotestyourself/gotestyourself/fs"
	"github.com/gotestyourself/gotestyourself/golden"
	"github.com/gotestyourself/gotestyourself/icmd"

	"github.com/docker/lunchbox/utils"
)

type registry struct {
	port      int
	container string
}

func startRegistry() (*registry, error) {
	r := &registry{}
	err := r.Start()
	return r, err
}

// Start starts a new docker registry on a random port
func (r *registry) Start() error {
	cmd := exec.Command("docker", "run", "--rm", "-d", "-P", "registry:2")
	output, err := cmd.Output()
	r.container = strings.Trim(string(output), " \r\n")
	return err
}

// Stop terminates this registry
func (r *registry) Stop() error {
	cmd := exec.Command("docker", "stop", r.container)
	_, err := cmd.CombinedOutput()
	return err
}

// Port returns the host port this registry listens on
func (r *registry) Port() (int, error) {
	if r.port != 0 {
		return r.port, nil
	}
	cmd := exec.Command("docker", "port", r.container, "5000")
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(string(output))
		return 0, err
	}
	sport := strings.Split(string(output), ":")[1]
	p, err := strconv.ParseInt(strings.Trim(sport, " \r\n"), 10, 32)
	if err == nil {
		r.port = int(p)
	}
	return r.port, err
}

var (
	dockerApp       = ""
	hasExperimental = false
	renderers       = ""
)

func getBinary(t *testing.T) (string, bool) {
	if dockerApp != "" {
		return dockerApp, hasExperimental
	}
	binName := findBinary()
	if binName == "" {
		t.Error("cannot locate docker-app binary")
	}
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

func findBinary() string {
	binNames := []string{
		os.Getenv("DOCKERAPP_BINARY"),
		"./docker-app-" + runtime.GOOS + binExt(),
		"./docker-app" + binExt(),
		"../_build/docker-app-" + runtime.GOOS + binExt(),
		"../_build/docker-app" + binExt(),
	}
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

// just run a command discarding everything
func runCommand(exe string, args ...string) {
	cmd := exec.Command(exe, args...)
	cmd.CombinedOutput()
}

// Run command, assert it succeeds, return its output
func assertCommand(t *testing.T, exe string, args ...string) []byte {
	cmd := exec.Command(exe, args...)
	output, err := cmd.CombinedOutput()
	assert.NilError(t, err, string(output))
	return output
}

func assertCommandOutput(t *testing.T, goldenFile string, cmd string, args ...string) {
	output := assertCommand(t, cmd, args...)
	golden.Assert(t, string(output), goldenFile)
}

func TestRenderBinary(t *testing.T) {
	getBinary(t)
	apps, err := ioutil.ReadDir("render")
	assert.NilError(t, err, "unable to get apps")
	for _, app := range apps {
		t.Log("testing", app.Name())
		if !checkRenderers(app.Name(), renderers) {
			t.Log("Required renderer not enabled.")
			continue
		}
		settings, overrides, env := gather(t, filepath.Join("render", app.Name()))
		args := []string{
			"render",
			filepath.Join("render", app.Name()),
		}
		for _, s := range settings {
			args = append(args, "-f", s)
		}
		for _, c := range overrides {
			args = append(args, "-c", c)
		}
		for k, v := range env {
			args = append(args, "-s", fmt.Sprintf("%s=%s", k, v))
		}
		t.Logf("executing with %v", args)
		cmd := exec.Command(dockerApp, args...)
		output, err := cmd.CombinedOutput()
		checkResult(t, string(output), err, filepath.Join("render", app.Name()))
	}
}

func randomName(prefix string) string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}
	return prefix + hex.EncodeToString(b)
}

func TestInitBinary(t *testing.T) {
	getBinary(t)
	composeData := `services:
  nginx:
    image: nginx:${NGINX_VERSION}
    command: nginx $NGINX_ARGS
`
	meta := `version: 0.1.0
name: app_test
description: my cool app
repository_prefix: ""
maintainers:
- name: bob
  email: ""
- name: joe
  email: joe@joe.com
targets:
  swarm: true
  kubernetes: true
`
	envData := "# some comment\nNGINX_VERSION=latest"
	inputDir := randomName("app_input_")
	os.Mkdir(inputDir, 0755)
	ioutil.WriteFile(filepath.Join(inputDir, "docker-compose.yml"), []byte(composeData), 0644)
	ioutil.WriteFile(filepath.Join(inputDir, ".env"), []byte(envData), 0644)
	defer os.RemoveAll(inputDir)

	testAppName := "app_test"
	dirName := utils.DirNameFromAppName(testAppName)
	defer os.RemoveAll(dirName)

	args := []string{
		"init",
		testAppName,
		"-c",
		filepath.Join(inputDir, "docker-compose.yml"),
		"-d",
		"my cool app",
		"-m", "bob",
		"-m", "joe:joe@joe.com",
	}
	assertCommand(t, dockerApp, args...)
	manifest := fs.Expected(
		t,
		fs.WithMode(0755),
		fs.WithFile("metadata.yml", meta, fs.WithMode(0644)), // too many variables, cheating
		fs.WithFile("docker-compose.yml", composeData, fs.WithMode(0644)),
		fs.WithFile("settings.yml", "NGINX_ARGS: FILL ME\nNGINX_VERSION: latest\n", fs.WithMode(0644)),
	)

	assert.Assert(t, fs.Equal(dirName, manifest))

	// test single-file init
	args = []string{
		"init",
		"tac",
		"-c",
		filepath.Join(inputDir, "docker-compose.yml"),
		"-d",
		"my cool app",
		"-m", "bob",
		"-m", "joe:joe@joe.com",
		"-s",
	}
	assertCommand(t, dockerApp, args...)
	defer os.Remove("tac.dockerapp")
	appData, _ := ioutil.ReadFile("tac.dockerapp")
	golden.Assert(t, string(appData), "init-singlefile.dockerapp")
}

func TestInspectBinary(t *testing.T) {
	dockerApp, _ := getBinary(t)
	assertCommandOutput(t, "envvariables-inspect.golden", dockerApp, "inspect", "render/envvariables")
}

func TestPackBinary(t *testing.T) {
	dockerApp, _ := getBinary(t)
	tempDir, err := ioutil.TempDir("", "dockerapp")
	assert.NilError(t, err)
	defer os.RemoveAll(tempDir)
	result := icmd.RunCommand(dockerApp, "pack", "helm", "-o", filepath.Join(tempDir, "test.dockerapp"))
	result.Assert(t, icmd.Success)
	// check that our commands run on the packed version
	result = icmd.RunCommand(dockerApp, "inspect", filepath.Join(tempDir, "test"))
	result.Assert(t, icmd.Success)
	assert.Assert(t, strings.Contains(result.Stdout(), "myapp"), "got: %s", result.Stdout())
	result = icmd.RunCommand(dockerApp, "render", filepath.Join(tempDir, "test"))
	result.Assert(t, icmd.Success)
	assert.Assert(t, strings.Contains(result.Stdout(), "nginx"))
	cwd, err := os.Getwd()
	assert.NilError(t, err)
	os.Chdir(tempDir)
	result = icmd.RunCommand(dockerApp, "helm", "test")
	result.Assert(t, icmd.Success)
	_, err = os.Stat("test.chart/Chart.yaml")
	assert.NilError(t, err)
	os.Mkdir("output", 0755)
	result = icmd.RunCommand(dockerApp, "unpack", "test", "-o", "output")
	result.Assert(t, icmd.Success)
	_, err = os.Stat("output/test.dockerapp/docker-compose.yml")
	assert.NilError(t, err)
	os.Chdir(cwd)
}

func TestHelmBinary(t *testing.T) {
	dockerApp, _ := getBinary(t)
	assertCommand(t, dockerApp, "helm", "helm", "-s", "myapp.nginx_version=2")
	chart, _ := ioutil.ReadFile("helm.chart/Chart.yaml")
	values, _ := ioutil.ReadFile("helm.chart/values.yaml")
	stack, _ := ioutil.ReadFile("helm.chart/templates/stack.yaml")
	golden.Assert(t, string(chart), "helm-expected.chart/Chart.yaml")
	golden.Assert(t, string(values), "helm-expected.chart/values.yaml")
	golden.Assert(t, string(stack), "helm-expected.chart/templates/stack.yaml")
}

func TestSplitMergeBinary(t *testing.T) {
	dockerApp, hasExperimental := getBinary(t)
	if !hasExperimental {
		t.Skip("experimental mode needed for this test")
	}
	app := "render/envvariables"
	assertCommand(t, dockerApp, "merge", app, "-o", "remerged.dockerapp")
	defer os.Remove("remerged.dockerapp")
	// test that inspect works on single-file
	assertCommandOutput(t, "envvariables-inspect.golden", dockerApp, "inspect", "remerged")
	// split it
	assertCommand(t, dockerApp, "split", "remerged", "-o", "splitted.dockerapp")
	defer os.RemoveAll("splitted.dockerapp")
	assertCommandOutput(t, "envvariables-inspect.golden", dockerApp, "inspect", "splitted")
}

func TestImageBinary(t *testing.T) {
	dockerApp, _ := getBinary(t)
	r, err := startRegistry()
	assert.NilError(t, err)
	defer r.Stop()
	port, err := r.Port()
	assert.NilError(t, err)
	registry := fmt.Sprintf("localhost:%v", port)
	defer func() {
		// no way to match both in one command
		cmd1 := exec.Command("docker", "image", "ls", "--format", "{{.ID}}", "--filter", "reference=*/*envvariables*")
		o1, _ := cmd1.Output()
		cmd2 := exec.Command("docker", "image", "ls", "--format", "{{.ID}}", "--filter", "reference=*/*/*envvariables*")
		o2, _ := cmd2.Output()
		refs := strings.Split(string(append(o1, o2...)), "\n")
		args := []string{"image", "rm", "-f"}
		args = append(args, refs...)
		runCommand("docker", args...)
	}()
	// save with tag/prefix override
	assertCommand(t, dockerApp, "save", "-t", "mytag", "-p", registry+"/myuser/", "render/envvariables")
	assertCommandOutput(t, "image-inspect-labels.golden", "docker", "inspect", "-f", "{{.Config.Labels.maintainers}}", registry+"/myuser/envvariables.dockerapp:mytag")
	// save with tag/prefix from metadata
	assertCommand(t, dockerApp, "save", "render/envvariables")
	assertCommandOutput(t, "image-inspect-labels.golden", "docker", "inspect", "-f", "{{.Config.Labels.maintainers}}", "alice/envvariables.dockerapp:0.1.0")
	// push to a registry
	assertCommand(t, dockerApp, "push", "-p", registry+"/myuser/", "render/envvariables")
	// various commands from an image
	assertCommand(t, dockerApp, "inspect", "alice/envvariables:0.1.0")
	assertCommand(t, dockerApp, "inspect", "alice/envvariables.dockerapp:0.1.0")
}
