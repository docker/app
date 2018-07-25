package e2e

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/docker/app/internal"
	"github.com/docker/app/internal/types"
	yaml "gopkg.in/yaml.v2"

	"gotest.tools/assert"
	"gotest.tools/fs"
	"gotest.tools/golden"
	"gotest.tools/icmd"
)

// just run a command discarding everything
func runCommand(exe string, args ...string) {
	cmd := exec.Command(exe, args...)
	cmd.CombinedOutput()
}

// Run command, assert it succeeds, return its output
func assertCommand(t *testing.T, exe string, args ...string) []byte {
	t.Helper()
	cmd := exec.Command(exe, args...)
	output, err := cmd.CombinedOutput()
	assert.NilError(t, err, string(output))
	return output
}

func assertCommandOutput(t *testing.T, goldenFile string, cmd string, args ...string) {
	t.Helper()
	output := assertCommand(t, cmd, args...)
	golden.Assert(t, string(output), goldenFile)
}

func assertCommandFailureOutput(t *testing.T, goldenFile string, exe string, args ...string) {
	t.Helper()
	cmd := exec.Command(exe, args...)
	output, err := cmd.CombinedOutput()
	assert.Assert(t, err != nil)
	golden.Assert(t, string(output), goldenFile)
}

func TestRenderBinary(t *testing.T) {
	getDockerAppBinary(t)
	apps, err := ioutil.ReadDir("render")
	assert.NilError(t, err, "unable to get apps")
	for _, app := range apps {
		if app.Name() == "testdata" {
			continue
		}
		t.Log("testing", app.Name())
		envs := []string{}
		if !checkRenderers(app.Name(), renderers) {
			t.Log("Required renderer not enabled.")
			continue
		} else if strings.HasPrefix(app.Name(), "template-") {
			envs = append(envs, "DOCKERAPP_RENDERERS="+strings.TrimPrefix(strings.TrimSuffix(app.Name(), ".dockerapp"), "template-"))
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
		t.Logf("executing with %v (envs %v)", args, envs)
		cmd := exec.Command(dockerApp, args...)
		cmd.Env = envs
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
	getDockerAppBinary(t)
	composeData := `version: "3.2"
services:
  nginx:
    image: nginx:${NGINX_VERSION}
    command: nginx $NGINX_ARGS
`
	meta := `# Version of the application
version: 0.1.0
# Name of the application
name: app-test
# A short description of the application
description: my cool app
# Namespace to use when pushing to a registry. This is typically your Hub username.
#namespace: myHubUsername
# List of application maintainers with name and email for each
maintainers:
  - name: bob
    email: 
  - name: joe
    email: joe@joe.com
`
	envData := "# some comment\nNGINX_VERSION=latest"
	inputDir := randomName("app_input_")
	os.Mkdir(inputDir, 0755)
	ioutil.WriteFile(filepath.Join(inputDir, internal.ComposeFileName), []byte(composeData), 0644)
	ioutil.WriteFile(filepath.Join(inputDir, ".env"), []byte(envData), 0644)
	defer os.RemoveAll(inputDir)

	testAppName := "app-test"
	dirName := internal.DirNameFromAppName(testAppName)
	defer os.RemoveAll(dirName)

	args := []string{
		"init",
		testAppName,
		"-c",
		filepath.Join(inputDir, internal.ComposeFileName),
		"-d",
		"my cool app",
		"-m", "bob",
		"-m", "joe:joe@joe.com",
	}
	assertCommand(t, dockerApp, args...)
	manifest := fs.Expected(
		t,
		fs.WithMode(0755),
		fs.WithFile(internal.MetadataFileName, meta, fs.WithMode(0644)), // too many variables, cheating
		fs.WithFile(internal.ComposeFileName, composeData, fs.WithMode(0644)),
		fs.WithFile(internal.SettingsFileName, "NGINX_ARGS: FILL ME\nNGINX_VERSION: latest\n", fs.WithMode(0644)),
	)
	assert.Assert(t, fs.Equal(dirName, manifest))

	// validate metadata with JSON Schema
	assertCommand(t, dockerApp, "validate", testAppName)

	// test single-file init
	args = []string{
		"init",
		"tac",
		"-c",
		filepath.Join(inputDir, internal.ComposeFileName),
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
	// Check various commands work on single-file app package
	assertCommand(t, dockerApp, "inspect", "tac")
	assertCommand(t, dockerApp, "render", "tac")
}

func TestDetectAppBinary(t *testing.T) {
	dockerApp, _ := getDockerAppBinary(t)
	// cwd = e2e
	assertCommand(t, dockerApp, "inspect")
	cwd, err := os.Getwd()
	assert.NilError(t, err)
	defer os.Chdir(cwd)
	os.Chdir("helm.dockerapp")
	assertCommand(t, dockerApp, "inspect")
	assertCommand(t, dockerApp, "inspect", ".")
	os.Chdir(filepath.Join(cwd, "render"))
	assertCommandFailureOutput(t, "inspect-multiple-apps.golden", dockerApp, "inspect")
}

func TestPackBinary(t *testing.T) {
	dockerApp, hasExperimental := getDockerAppBinary(t)
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

func runHelmCommand(t *testing.T, args ...string) *fs.Dir {
	t.Helper()
	dockerApp, _ := getDockerAppBinary(t)
	abs, err := filepath.Abs(".")
	assert.NilError(t, err)
	dir := fs.NewDir(t, t.Name(), fs.FromDir(abs))
	result := icmd.RunCmd(icmd.Cmd{
		Command: append([]string{dockerApp}, args...),
		Dir:     dir.Path(),
	})
	result.Assert(t, icmd.Success)
	return dir
}

func TestHelmBinary(t *testing.T) {
	dir := runHelmCommand(t, "helm", "helm", "-s", "myapp.nginx_version=2")
	defer dir.Remove()

	chart, _ := ioutil.ReadFile(dir.Join("helm.chart/Chart.yaml"))
	values, _ := ioutil.ReadFile(dir.Join("helm.chart/values.yaml"))
	stack, _ := ioutil.ReadFile(dir.Join("helm.chart/templates/stack.yaml"))
	golden.Assert(t, string(chart), "helm-expected.chart/Chart.yaml", "chart file is wrong")
	golden.Assert(t, string(values), "helm-expected.chart/values.yaml", "values file is wrong")
	golden.Assert(t, string(stack), "helm-expected.chart/templates/stack.yaml", "stack file is wrong")
}

func TestHelmV1Beta1Binary(t *testing.T) {
	dir := runHelmCommand(t, "helm", "helm", "-s", "myapp.nginx_version=2", "--stack-version", "v1beta1")
	defer dir.Remove()

	chart, _ := ioutil.ReadFile(dir.Join("helm.chart/Chart.yaml"))
	values, _ := ioutil.ReadFile(dir.Join("helm.chart/values.yaml"))
	stack, _ := ioutil.ReadFile(dir.Join("helm.chart/templates/stack.yaml"))
	golden.Assert(t, string(chart), "helm-expected.chart/Chart.yaml", "chart file is wrong")
	golden.Assert(t, string(values), "helm-expected.chart/values.yaml", "values file is wrong")
	golden.Assert(t, string(stack), "helm-expected.chart/templates/stack-v1beta1.yaml", "stack file is wrong")
}

func TestHelmInvalidStackVersionBinary(t *testing.T) {
	dockerApp, _ := getDockerAppBinary(t)
	assertCommandFailureOutput(t, "invalid-stack-version.golden", dockerApp, "helm", "helm", "--stack-version", "foobar")
}

func TestSplitMergeBinary(t *testing.T) {
	dockerApp, _ := getDockerAppBinary(t)
	app := "render/envvariables"
	assertCommand(t, dockerApp, "merge", app, "-o", "remerged.dockerapp")
	defer os.Remove("remerged.dockerapp")
	// test that inspect works on single-file
	assertCommandOutput(t, "envvariables-inspect.golden", dockerApp, "inspect", "remerged")
	// split it
	assertCommand(t, dockerApp, "split", "remerged", "-o", "split.dockerapp")
	defer os.RemoveAll("split.dockerapp")
	assertCommandOutput(t, "envvariables-inspect.golden", dockerApp, "inspect", "split")
	// test inplace
	assertCommand(t, dockerApp, "merge", "split")
	assertCommand(t, dockerApp, "split", "split")
}

func TestImageBinary(t *testing.T) {
	dockerApp, _ := getDockerAppBinary(t)
	r := startRegistry(t)
	defer r.stop(t)
	registry := r.getAddress(t)
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
	assertCommand(t, dockerApp, "save", "-t", "mytag", "--namespace", registry+"/myuser", "render/envvariables")
	assertCommandOutput(t, "image-inspect-labels.golden", "docker", "inspect", "-f", "{{.Config.Labels.maintainers}}", registry+"/myuser/envvariables.dockerapp:mytag")
	// save with tag/prefix from metadata
	assertCommand(t, dockerApp, "save", "render/envvariables")
	assertCommandOutput(t, "image-inspect-labels.golden", "docker", "inspect", "-f", "{{.Config.Labels.maintainers}}", "alice/envvariables.dockerapp:0.1.0")
	// push to a registry
	assertCommand(t, dockerApp, "push", "--namespace", registry+"/myuser", "render/envvariables")
	assertCommand(t, dockerApp, "push", "--namespace", registry+"/myuser", "-t", "latest", "render/envvariables")
	assertCommand(t, "docker", "image", "rm", registry+"/myuser/envvariables.dockerapp:0.1.0")
	assertCommand(t, dockerApp, "inspect", registry+"/myuser/envvariables.dockerapp:0.1.0")
	assertCommand(t, dockerApp, "inspect", registry+"/myuser/envvariables.dockerapp")
	assertCommand(t, dockerApp, "inspect", registry+"/myuser/envvariables")
	assertCommand(t, dockerApp, "inspect", registry+"/myuser/envvariables:0.1.0")
	// various commands from an image
	assertCommand(t, dockerApp, "inspect", "alice/envvariables:0.1.0")
	assertCommand(t, dockerApp, "inspect", "alice/envvariables.dockerapp:0.1.0")
}

func TestForkBinary(t *testing.T) {
	dockerApp, _ := getDockerAppBinary(t)
	r := startRegistry(t)
	defer r.stop(t)
	registry := r.getAddress(t)
	assertCommand(t, dockerApp, "save", "--namespace", registry+"/acmecorp", "fork/simple")
	assertCommand(t, dockerApp, "push", "--namespace", registry+"/acmecorp", "fork/simple")

	tempDir, err := ioutil.TempDir("", "dockerapptest")
	assert.NilError(t, err)
	defer os.RemoveAll(tempDir)

	assertCommand(t, dockerApp, "fork", registry+"/acmecorp/simple.dockerapp:1.1.0-beta1", "acmecorp/scarlet.devil", "-p", tempDir, "-m", "Remilia Scarlet:remilia@acmecorp.cool")
	metadata, err := ioutil.ReadFile(filepath.Join(tempDir, "scarlet.devil.dockerapp", "metadata.yml"))
	assert.NilError(t, err)

	golden.Assert(t, string(metadata), "expected-fork-metadata.golden")
	var decodedMeta types.AppMetadata
	err = yaml.Unmarshal(metadata, &decodedMeta)
	assert.NilError(t, err)
	var expected = types.AppMetadata{
		Name:        "scarlet.devil",
		Namespace:   "acmecorp",
		Version:     "1.1.0-beta1",
		Description: "new fancy webapp with microservices",
		Maintainers: types.Maintainers{
			{Name: "Remilia Scarlet", Email: "remilia@acmecorp.cool"},
		},
		Parents: types.Parents{
			{
				Name:      "simple",
				Namespace: "acmecorp",
				Version:   "1.1.0-beta1",
				Maintainers: types.Maintainers{
					{Name: "John Developer", Email: "john.dev@acmecorp.cool"},
					{Name: "Jane Developer", Email: "jane.dev@acmecorp.cool"},
				},
			},
		},
	}

	assert.DeepEqual(t, decodedMeta, expected)
}
