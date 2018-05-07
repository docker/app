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
	"strings"
	"testing"

	"github.com/gotestyourself/gotestyourself/assert"
	"github.com/gotestyourself/gotestyourself/fs"
	"github.com/gotestyourself/gotestyourself/golden"
	"github.com/gotestyourself/gotestyourself/icmd"

	"github.com/docker/lunchbox/utils"
)

var (
	dockerApp       = ""
	hasExperimental = false
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
	hasExperimental = strings.Contains(string(output), "Experimental: on")
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

func TestRenderBinary(t *testing.T) {
	getBinary(t)
	apps, err := ioutil.ReadDir("render")
	assert.NilError(t, err, "unable to get apps")
	for _, app := range apps {
		t.Log("testing", app.Name())
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
	envData := "# some comment\nNGINX_VERSION=latest"
	inputDir := randomName("app_input_")
	os.Mkdir(inputDir, 0755)
	ioutil.WriteFile(filepath.Join(inputDir, "docker-compose.yml"), []byte(composeData), 0644)
	ioutil.WriteFile(filepath.Join(inputDir, ".env"), []byte(envData), 0644)
	defer os.RemoveAll(inputDir)

	testAppName := randomName("app_")
	dirName := utils.DirNameFromAppName(testAppName)
	defer os.RemoveAll(dirName)

	args := []string{
		"init",
		testAppName,
		"-c",
		filepath.Join(inputDir, "docker-compose.yml"),
	}
	cmd := exec.Command(dockerApp, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(output)
	}
	assert.NilError(t, err)
	meta, err := ioutil.ReadFile(filepath.Join(dirName, "metadata.yml"))
	assert.NilError(t, err)
	manifest := fs.Expected(
		t,
		fs.WithMode(0755),
		fs.WithFile("metadata.yml", string(meta), fs.WithMode(0644)), // too many variables, cheating
		fs.WithFile("docker-compose.yml", composeData, fs.WithMode(0644)),
		fs.WithFile("settings.yml", "NGINX_ARGS: FILL ME\nNGINX_VERSION: latest\n", fs.WithMode(0644)),
	)

	assert.Assert(t, fs.Equal(dirName, manifest))
}

func TestPackBinary(t *testing.T) {
	dockerApp, hasExperimental := getBinary(t)
	if !hasExperimental {
		t.Skip("experimental mode needed for this test")
	}
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
	cmd := exec.Command(dockerApp, "helm", "helm", "-s", "myapp.nginx_version=2")
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(string(output))
	}
	assert.NilError(t, err)
	chart, _ := ioutil.ReadFile("helm.chart/Chart.yaml")
	values, _ := ioutil.ReadFile("helm.chart/values.yaml")
	stack, _ := ioutil.ReadFile("helm.chart/templates/stack.yaml")
	golden.AssertBytes(t, chart, "helm-expected.chart/Chart.yaml")
	golden.AssertBytes(t, values, "helm-expected.chart/values.yaml")
	golden.AssertBytes(t, stack, "helm-expected.chart/templates/stack.yaml")
}
