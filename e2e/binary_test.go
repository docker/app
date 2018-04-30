package e2e

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"runtime"
	"testing"

	"github.com/gotestyourself/gotestyourself/assert"
	"github.com/gotestyourself/gotestyourself/fs"

	"github.com/docker/lunchbox/utils"
)

var (
	dockerApp = ""
)

func getBinary(t *testing.T) string {
	if dockerApp != "" {
		return dockerApp
	}
	dockerApp = os.Getenv("DOCKERAPP_BINARY")
	if dockerApp == "" {
		binName := "docker-app"
		if runtime.GOOS == "windows" {
			binName += ".exe"
		}
		locations := []string{".", "../_build"}
		for _, l := range locations {
			b := path.Join(l, binName)
			if _, err := os.Stat(b); err == nil {
				dockerApp = b
				break
			}
		}
	}
	if dockerApp == "" {
		t.Error("cannot locate docker-app binary")
	}
	cmd := exec.Command(dockerApp, "version")
	_, err := cmd.CombinedOutput()
	assert.NilError(t, err, "failed to execute docker-app binary")
	return dockerApp
}

func TestRenderBinary(t *testing.T) {
	getBinary(t)
	apps, err := ioutil.ReadDir("render")
	assert.NilError(t, err, "unable to get apps")
	for _, app := range apps {
		t.Log("testing", app.Name())
		settings, overrides, env := gather(t, path.Join("render", app.Name()))
		args := []string{
			"render",
			path.Join("render", app.Name()),
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
		checkResult(t, string(output), err, path.Join("render", app.Name()))
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
	ioutil.WriteFile(path.Join(inputDir, "docker-compose.yml"), []byte(composeData), 0644)
	ioutil.WriteFile(path.Join(inputDir, ".env"), []byte(envData), 0644)
	defer os.RemoveAll(inputDir)

	testAppName := randomName("app_")
	dirName := utils.DirNameFromAppName(testAppName)
	defer os.RemoveAll(dirName)

	args := []string{
		"init",
		testAppName,
		"-c",
		path.Join(inputDir, "docker-compose.yml"),
	}
	cmd := exec.Command(dockerApp, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(string(output))
	}
	assert.NilError(t, err)
	meta, err := ioutil.ReadFile(path.Join(dirName, "metadata.yml"))
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

func TestHelmBinary(t *testing.T) {
	getBinary(t)
	cmd := exec.Command(dockerApp, "helm", "helm", "-t", "-s", "myapp.nginx_version=2")
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(string(output))
	}
	assert.NilError(t, err)
	chart, _ := ioutil.ReadFile("helm-expected.chart/Chart.yaml")
	values, _ := ioutil.ReadFile("helm-expected.chart/values.yaml")
	stack, _ := ioutil.ReadFile("helm-expected.chart/templates/stack.yaml")
	manifest := fs.Expected(
		t,
		fs.WithMode(0755),
		fs.WithFile("Chart.yaml", string(chart), fs.WithMode(0644)),
		fs.WithFile("values.yaml", string(values), fs.WithMode(0644)),
		fs.WithDir("templates",
			fs.WithMode(0755),
			fs.WithFile("stack.yaml", string(stack), fs.WithMode(0644)),
		),
	)
	assert.Assert(t, fs.Equal("helm.chart", manifest))
}
