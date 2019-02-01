package e2e

import (
	"fmt"
	"net"
	"strconv"
	"time"

	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/docker/app/internal"
	"github.com/docker/app/internal/yaml"
	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
	"gotest.tools/fs"
	"gotest.tools/golden"
	"gotest.tools/icmd"
	"gotest.tools/poll"
	"gotest.tools/skip"
)

const (
	singleFileApp = `version: 0.1.0
name: helloworld
description: "hello world app"
namespace: "foo"
---
version: '3.5'
services:
  hello-world:
    image: hello-world
---
# This section contains the default values for your application parameters.`
	targetName       = "testTargetContext"
	installationName = "test-simple-installation"
)

func TestRenderTemplates(t *testing.T) {
	skip.If(t, !hasExperimental, "experimental mode needed for this test")
	appsPath := filepath.Join("testdata", "templates")
	apps, err := ioutil.ReadDir(appsPath)
	assert.NilError(t, err, "unable to get apps")
	for _, app := range apps {
		appPath := filepath.Join(appsPath, app.Name())
		if !checkRenderers(app.Name(), renderers) {
			t.Log("Required renderer not enabled")
			continue
		}
		t.Run(app.Name(), testRenderApp(appPath, "DOCKERAPP_RENDERERS="+app.Name()))
	}
}

func TestRender(t *testing.T) {
	appsPath := filepath.Join("testdata", "render")
	apps, err := ioutil.ReadDir(appsPath)
	assert.NilError(t, err, "unable to get apps")
	for _, app := range apps {
		appPath := filepath.Join(appsPath, app.Name())
		t.Run(app.Name(), testRenderApp(appPath))
	}
}

func testRenderApp(appPath string, env ...string) func(*testing.T) {
	return func(t *testing.T) {
		envParameters := map[string]string{}
		data, err := ioutil.ReadFile(filepath.Join(appPath, "env.yml"))
		assert.NilError(t, err)
		assert.NilError(t, yaml.Unmarshal(data, &envParameters))
		args := []string{dockerApp, "render", filepath.Join(appPath, "my.dockerapp"),
			"-f", filepath.Join(appPath, "parameters-0.yml"),
		}
		for k, v := range envParameters {
			args = append(args, "-s", fmt.Sprintf("%s=%s", k, v))
		}
		result := icmd.RunCmd(icmd.Cmd{
			Command: args,
			Env:     env,
		}).Assert(t, icmd.Success)
		assert.Assert(t, is.Equal(readFile(t, filepath.Join(appPath, "expected.txt")), result.Stdout()), "rendering mismatch")
	}
}

func TestRenderFormatters(t *testing.T) {
	appPath := filepath.Join("testdata", "simple", "simple.dockerapp")
	result := icmd.RunCommand(dockerApp, "render", "--formatter", "json", appPath).Assert(t, icmd.Success)
	golden.Assert(t, result.Stdout(), "expected-json-render.golden")

	result = icmd.RunCommand(dockerApp, "render", "--formatter", "yaml", appPath).Assert(t, icmd.Success)
	golden.Assert(t, result.Stdout(), "expected-yaml-render.golden")
}

func TestInit(t *testing.T) {
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
#namespace: myhubusername
# List of application maintainers with name and email for each
maintainers:
  - name: bob
    email: 
  - name: joe
    email: joe@joe.com
`
	envData := "# some comment\nNGINX_VERSION=latest"
	dir := fs.NewDir(t, "app_input",
		fs.WithFile(internal.ComposeFileName, composeData),
		fs.WithFile(".env", envData),
	)
	defer dir.Remove()

	testAppName := "app-test"
	dirName := internal.DirNameFromAppName(testAppName)
	defer os.RemoveAll(dirName)

	icmd.RunCommand(dockerApp, "init", testAppName,
		"-c", dir.Join(internal.ComposeFileName),
		"-d", "my cool app",
		"-m", "bob",
		"-m", "joe:joe@joe.com",
	).Assert(t, icmd.Success)
	manifest := fs.Expected(
		t,
		fs.WithMode(0755),
		fs.WithFile(internal.MetadataFileName, meta, fs.WithMode(0644)), // too many variables, cheating
		fs.WithFile(internal.ComposeFileName, composeData, fs.WithMode(0644)),
		fs.WithFile(internal.ParametersFileName, "NGINX_ARGS: FILL ME\nNGINX_VERSION: latest\n", fs.WithMode(0644)),
	)
	assert.Assert(t, fs.Equal(dirName, manifest))

	// validate metadata with JSON Schema
	icmd.RunCommand(dockerApp, "validate", testAppName).Assert(t, icmd.Success)

	// test single-file init
	icmd.RunCommand(dockerApp, "init", "tac",
		"-c", dir.Join(internal.ComposeFileName),
		"-d", "my cool app",
		"-m", "bob",
		"-m", "joe:joe@joe.com",
		"-s",
	).Assert(t, icmd.Success)
	defer os.Remove("tac.dockerapp")
	appData, err := ioutil.ReadFile("tac.dockerapp")
	assert.NilError(t, err)
	golden.AssertBytes(t, appData, "init-singlefile.dockerapp")
	// Check various commands work on single-file app package
	icmd.RunCommand(dockerApp, "inspect", "tac").Assert(t, icmd.Success)
	icmd.RunCommand(dockerApp, "render", "tac").Assert(t, icmd.Success)
}

func TestDetectApp(t *testing.T) {
	// cwd = e2e
	dir := fs.NewDir(t, "detect-app-binary",
		fs.WithDir("attachments.dockerapp", fs.FromDir("testdata/attachments.dockerapp")),
		fs.WithDir("render",
			fs.WithDir("app1.dockerapp", fs.FromDir("testdata/render/envvariables/my.dockerapp")),
			fs.WithDir("app2.dockerapp", fs.FromDir("testdata/render/envvariables/my.dockerapp")),
		),
	)
	defer dir.Remove()
	icmd.RunCmd(icmd.Cmd{
		Command: []string{dockerApp, "inspect"},
		Dir:     dir.Path(),
	}).Assert(t, icmd.Success)
	icmd.RunCmd(icmd.Cmd{
		Command: []string{dockerApp, "inspect"},
		Dir:     dir.Join("attachments.dockerapp"),
	}).Assert(t, icmd.Success)
	icmd.RunCmd(icmd.Cmd{
		Command: []string{dockerApp, "inspect", "."},
		Dir:     dir.Join("attachments.dockerapp"),
	}).Assert(t, icmd.Success)
	result := icmd.RunCmd(icmd.Cmd{
		Command: []string{dockerApp, "inspect"},
		Dir:     dir.Join("render"),
	})
	result.Assert(t, icmd.Expected{
		ExitCode: 1,
		Err:      "Error: multiple applications found in current directory, specify the application name on the command line",
	})
}

func TestPack(t *testing.T) {
	skip.If(t, !hasExperimental, "experimental mode needed for this test")
	tempDir, err := ioutil.TempDir("", "dockerapp")
	assert.NilError(t, err)
	defer os.RemoveAll(tempDir)
	icmd.RunCommand(dockerApp, "pack", "testdata/attachments", "-o", filepath.Join(tempDir, "test.dockerapp")).Assert(t, icmd.Success)
	// check that our commands run on the packed version
	icmd.RunCommand(dockerApp, "inspect", filepath.Join(tempDir, "test")).Assert(t, icmd.Expected{
		Out: "myapp",
	})
	icmd.RunCommand(dockerApp, "render", filepath.Join(tempDir, "test")).Assert(t, icmd.Expected{
		Out: "nginx",
	})
	assert.NilError(t, os.Mkdir(filepath.Join(tempDir, "output"), 0755))
	icmd.RunCmd(icmd.Cmd{
		Command: []string{dockerApp, "unpack", "test", "-o", "output"},
		Dir:     tempDir,
	}).Assert(t, icmd.Success)
	_, err = os.Stat(filepath.Join(tempDir, "output", "test.dockerapp", "docker-compose.yml"))
	assert.NilError(t, err)
}

func TestSplitMerge(t *testing.T) {
	icmd.RunCommand(dockerApp, "merge", "testdata/render/envvariables/my.dockerapp", "-o", "remerged.dockerapp").Assert(t, icmd.Success)
	defer os.Remove("remerged.dockerapp")
	// test that inspect works on single-file
	result := icmd.RunCommand(dockerApp, "inspect", "remerged").Assert(t, icmd.Success)
	golden.Assert(t, result.Combined(), "envvariables-inspect.golden")
	// split it
	icmd.RunCommand(dockerApp, "split", "remerged", "-o", "split.dockerapp").Assert(t, icmd.Success)
	defer os.RemoveAll("split.dockerapp")
	result = icmd.RunCommand(dockerApp, "inspect", "remerged").Assert(t, icmd.Success)
	golden.Assert(t, result.Combined(), "envvariables-inspect.golden")
	// test inplace
	icmd.RunCommand(dockerApp, "merge", "split").Assert(t, icmd.Success)
	icmd.RunCommand(dockerApp, "split", "split").Assert(t, icmd.Success)
}

func TestURL(t *testing.T) {
	url := "https://raw.githubusercontent.com/docker/app/v0.4.1/examples/hello-world/hello-world.dockerapp"
	result := icmd.RunCommand(dockerApp, "inspect", url).Assert(t, icmd.Success)
	assert.Assert(t, golden.String(result.Combined(), "helloworld-inspect.golden"))
}

func TestWithRegistry(t *testing.T) {
	r := startRegistry(t)
	defer r.Stop(t)
	registry := r.GetAddress(t)
	// push to a registry
	icmd.RunCommand(dockerApp, "push", "--namespace", registry+"/myuser", "testdata/render/envvariables/my.dockerapp").Assert(t, icmd.Success)
	icmd.RunCommand(dockerApp, "push", "--namespace", registry+"/myuser", "-t", "latest", "testdata/render/envvariables/my.dockerapp").Assert(t, icmd.Success)
	icmd.RunCommand(dockerApp, "inspect", registry+"/myuser/my.dockerapp:0.1.0").Assert(t, icmd.Success)
	icmd.RunCommand(dockerApp, "inspect", registry+"/myuser/my.dockerapp").Assert(t, icmd.Success)
	icmd.RunCommand(dockerApp, "inspect", registry+"/myuser/my").Assert(t, icmd.Success)
	icmd.RunCommand(dockerApp, "inspect", registry+"/myuser/my:0.1.0").Assert(t, icmd.Success)
	// push a single-file app to a registry
	dir := fs.NewDir(t, "save-prepare-build", fs.WithFile("my.dockerapp", singleFileApp))
	defer dir.Remove()
	icmd.RunCommand(dockerApp, "push", "--namespace", registry+"/myuser", dir.Join("my.dockerapp")).Assert(t, icmd.Success)

	// push with custom repo name
	icmd.RunCommand(dockerApp, "push", "-t", "marshmallows", "--namespace", registry+"/rainbows", "--repo", "unicorns", "testdata/render/envvariables/my.dockerapp").Assert(t, icmd.Success)
	icmd.RunCommand(dockerApp, "inspect", registry+"/rainbows/unicorns:marshmallows").Assert(t, icmd.Success)
}

func TestAttachmentsWithRegistry(t *testing.T) {
	r := startRegistry(t)
	defer r.Stop(t)
	registry := r.GetAddress(t)

	dir := fs.NewDir(t, "testattachments",
		fs.WithDir("attachments.dockerapp", fs.FromDir("testdata/attachments.dockerapp")),
	)
	defer dir.Remove()

	icmd.RunCommand(dockerApp, "push", "--namespace", registry+"/acmecorp", dir.Join("attachments.dockerapp")).Assert(t, icmd.Success)

	// inspect will run the core pull code too
	result := icmd.RunCommand(dockerApp, "inspect", registry+"/acmecorp/attachments.dockerapp:0.1.0")

	result.Assert(t, icmd.Success)
	resultOutput := result.Combined()

	assert.Assert(t, strings.Contains(resultOutput, "config.cfg"))
	assert.Assert(t, strings.Contains(resultOutput, "nesteddir/config2.cfg"))
	assert.Assert(t, strings.Contains(resultOutput, "nesteddir/nested2/nested3/config3.cfg"))
}

func TestBundle(t *testing.T) {
	simpleDockerApp := filepath.Join("simple", "simple.dockerapp")
	composeContent := string(golden.Get(t, filepath.Join(simpleDockerApp, internal.ComposeFileName)))
	metadataContent := string(golden.Get(t, filepath.Join(simpleDockerApp, internal.MetadataFileName)))
	parametersContent := string(golden.Get(t, filepath.Join(simpleDockerApp, internal.ParametersFileName)))

	dir := fs.NewDir(t, "test-bundle",
		fs.WithDir("simple.dockerapp",
			fs.WithFile(internal.ComposeFileName, composeContent),
			fs.WithFile(internal.MetadataFileName, metadataContent),
			fs.WithFile(internal.ParametersFileName, parametersContent),
		),
	)
	defer dir.Remove()

	outputPath := dir.Join("bundle.json")
	icmd.RunCmd(icmd.Cmd{
		Command: []string{dockerApp, "bundle"},
		Dir:     dir.Path(),
	}).Assert(t, icmd.Success)

	golden.Assert(t, string(golden.Get(t, outputPath)), "simple-bundle.golden")
}

func TestLifeCycle(t *testing.T) {
	simpleDockerApp := filepath.Join("testdata", "echo.dockerapp")

	// Install
	ret := icmd.RunCommand(dockerApp, "install", simpleDockerApp,
		"--name", installationName,
		"--target-context", targetName)
	ret.Assert(t, icmd.Success)
	golden.Assert(t, ret.Stdout(), "install-test-simple-installation-success.golden")

	// Double install must fail
	ret = icmd.RunCommand(dockerApp, "install", simpleDockerApp,
		"--name", installationName,
		"--target-context", targetName)
	ret.Assert(t, icmd.Expected{ExitCode: 1})
	golden.Assert(t, ret.Stderr(), "install-test-simple-installation-already-exists.golden")

	// Status for 1 replica
	icmd.RunCommand(dockerApp, "status", installationName,
		"--target-context", targetName).
		Assert(t, icmd.Success)
	poll.WaitOn(t, checkStatusFn(t, "1/1"))
	poll.WaitOn(t, checkPort)

	// Upgrade
	ret = icmd.RunCommand(dockerApp, "upgrade", installationName,
		"--set", "echoapp.replicas=2",
		"--target-context", targetName).
		Assert(t, icmd.Success)

	// Status for 2 replicas
	icmd.RunCommand(dockerApp, "status", installationName,
		"--target-context", targetName).
		Assert(t, icmd.Success)
	poll.WaitOn(t, checkStatusFn(t, "2/2"))
	poll.WaitOn(t, checkPort)

	// Uninstall
	icmd.RunCommand(dockerApp, "uninstall", installationName,
		"--target-context", targetName).
		Assert(t, icmd.Success)
}

func checkStatusFn(t *testing.T, expectedReplicas string) func(t poll.LogT) poll.Result {
	return func(_ poll.LogT) poll.Result {
		var id, name, mode, replicas, image string
		ret := icmd.RunCommand(dockerApp, "status", installationName, "--target-context", targetName)
		ret.Assert(t, icmd.Success)
		headerAndValues := strings.Split(ret.Stdout(), "\n")
		assert.Assert(t, len(headerAndValues) > 1)
		_, err := fmt.Sscanf(headerAndValues[1], "%s %s %s %s %s", &id, &name, &mode, &replicas, &image)
		assert.NilError(t, err)
		if len(id) == 12 &&
			"test-simple-installation_hello" == name &&
			"replicated" == mode &&
			expectedReplicas == replicas &&
			"hashicorp/http-echo:latest" == image {
			return poll.Success()
		}
		return poll.Continue("failed to get the expected status")
	}
}

func checkPort(t poll.LogT) poll.Result {
	port := strconv.Itoa(8080)
	conn, err := net.DialTimeout("tcp", net.JoinHostPort("localhost", port), time.Second)
	if conn != nil {
		conn.Close()
	}
	if err != nil {
		return poll.Continue("could not connect to port '%s'", port)
	}
	return poll.Success()
}
