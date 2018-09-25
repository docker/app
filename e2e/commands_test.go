package e2e

import (
	"fmt"
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
# This section contains the default values for your application settings.`
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
		envSettings := map[string]string{}
		data, err := ioutil.ReadFile(filepath.Join(appPath, "env.yml"))
		assert.NilError(t, err)
		assert.NilError(t, yaml.Unmarshal(data, &envSettings))
		args := []string{dockerApp, "render", filepath.Join(appPath, "my.dockerapp"),
			"-f", filepath.Join(appPath, "settings-0.yml"),
		}
		for k, v := range envSettings {
			args = append(args, "-s", fmt.Sprintf("%s=%s", k, v))
		}
		result := icmd.RunCmd(icmd.Cmd{
			Command: args,
			Env:     env,
		}).Assert(t, icmd.Success)
		assert.Assert(t, is.Equal(readFile(t, filepath.Join(appPath, "expected.txt")), result.Stdout()), "rendering missmatch")
	}
}

func TestRenderFormatters(t *testing.T) {
	appPath := filepath.Join("testdata", "fork", "simple.dockerapp")
	result := icmd.RunCommand(dockerApp, "render", "--formatter", "json", appPath).Assert(t, icmd.Success)
	assert.Assert(t, golden.String(result.Stdout(), "expected-json-render.golden"))

	result = icmd.RunCommand(dockerApp, "render", "--formatter", "yaml", appPath).Assert(t, icmd.Success)
	assert.Assert(t, golden.String(result.Stdout(), "expected-yaml-render.golden"))
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
#namespace: myHubUsername
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
		fs.WithFile(internal.SettingsFileName, "NGINX_ARGS: FILL ME\nNGINX_VERSION: latest\n", fs.WithMode(0644)),
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
	assert.Assert(t, golden.Bytes(appData, "init-singlefile.dockerapp"))
	// Check various commands work on single-file app package
	icmd.RunCommand(dockerApp, "inspect", "tac").Assert(t, icmd.Success)
	icmd.RunCommand(dockerApp, "render", "tac").Assert(t, icmd.Success)
}

func TestDetectApp(t *testing.T) {
	// cwd = e2e
	dir := fs.NewDir(t, "detect-app-binary",
		fs.WithDir("helm.dockerapp", fs.FromDir("testdata/helm.dockerapp")),
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
		Dir:     dir.Join("helm.dockerapp"),
	}).Assert(t, icmd.Success)
	icmd.RunCmd(icmd.Cmd{
		Command: []string{dockerApp, "inspect", "."},
		Dir:     dir.Join("helm.dockerapp"),
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
	icmd.RunCommand(dockerApp, "pack", "testdata/helm", "-o", filepath.Join(tempDir, "test.dockerapp")).Assert(t, icmd.Success)
	// check that our commands run on the packed version
	icmd.RunCommand(dockerApp, "inspect", filepath.Join(tempDir, "test")).Assert(t, icmd.Expected{
		Out: "myapp",
	})
	icmd.RunCommand(dockerApp, "render", filepath.Join(tempDir, "test")).Assert(t, icmd.Expected{
		Out: "nginx",
	})
	icmd.RunCmd(icmd.Cmd{
		Command: []string{dockerApp, "helm", "test"},
		Dir:     tempDir,
	}).Assert(t, icmd.Success)
	_, err = os.Stat(filepath.Join(tempDir, "test.chart", "Chart.yaml"))
	assert.NilError(t, err)
	assert.NilError(t, os.Mkdir(filepath.Join(tempDir, "output"), 0755))
	icmd.RunCmd(icmd.Cmd{
		Command: []string{dockerApp, "unpack", "test", "-o", "output"},
		Dir:     tempDir,
	}).Assert(t, icmd.Success)
	_, err = os.Stat(filepath.Join(tempDir, "output", "test.dockerapp", "docker-compose.yml"))
	assert.NilError(t, err)
}

func TestHelm(t *testing.T) {
	t.Run("default", testHelm(""))
	t.Run("v1beta1", testHelm("v1beta1"))
	t.Run("v1beta2", testHelm("v1beta2"))
}

func testHelm(version string) func(*testing.T) {
	return func(t *testing.T) {
		dir := fs.NewDir(t, "testHelmBinary", fs.FromDir("testdata"))
		defer dir.Remove()
		cmd := []string{dockerApp, "helm", "helm", "-s", "myapp.nginx_version=2"}
		if version != "" {
			cmd = append(cmd, "--stack-version", version)
		}
		icmd.RunCmd(icmd.Cmd{
			Command: cmd,
			Dir:     dir.Path(),
		}).Assert(t, icmd.Success)

		chart := golden.Get(t, dir.Join("helm.chart/Chart.yaml"))
		values := golden.Get(t, dir.Join("helm.chart/values.yaml"))
		stack := golden.Get(t, dir.Join("helm.chart/templates/stack.yaml"))
		assert.Check(t, golden.String(string(chart), "helm-expected.chart/Chart.yaml"))
		assert.Check(t, golden.String(string(values), "helm-expected.chart/values.yaml"))
		assert.Check(t, golden.String(string(stack), "helm-expected.chart/templates/stack"+version+".yaml"))
	}
}

func TestHelmInvalidStackVersion(t *testing.T) {
	icmd.RunCommand(dockerApp, "helm", "testdata/helm", "--stack-version", "foobar").Assert(t, icmd.Expected{
		ExitCode: 1,
		Err:      `Error: invalid stack version "foobar" (accepted values: v1beta1, v1beta2)`,
	})
}

func TestSplitMerge(t *testing.T) {
	icmd.RunCommand(dockerApp, "merge", "testdata/render/envvariables/my.dockerapp", "-o", "remerged.dockerapp").Assert(t, icmd.Success)
	defer os.Remove("remerged.dockerapp")
	// test that inspect works on single-file
	result := icmd.RunCommand(dockerApp, "inspect", "remerged").Assert(t, icmd.Success)
	assert.Assert(t, golden.String(result.Combined(), "envvariables-inspect.golden"))
	// split it
	icmd.RunCommand(dockerApp, "split", "remerged", "-o", "split.dockerapp").Assert(t, icmd.Success)
	defer os.RemoveAll("split.dockerapp")
	result = icmd.RunCommand(dockerApp, "inspect", "remerged").Assert(t, icmd.Success)
	assert.Assert(t, golden.String(result.Combined(), "envvariables-inspect.golden"))
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
	t.Run("image", testImage(registry))
	t.Run("fork", testFork(registry))
}

func testImage(registry string) func(*testing.T) {
	return func(t *testing.T) {
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
}

func testFork(registry string) func(*testing.T) {
	return func(t *testing.T) {
		icmd.RunCommand(dockerApp, "push", "--namespace", registry+"/acmecorp", "testdata/fork/simple").Assert(t, icmd.Success)

		tempDir := fs.NewDir(t, "dockerapptest")
		defer tempDir.Remove()

		icmd.RunCommand(dockerApp, "fork", registry+"/acmecorp/simple.dockerapp:1.1.0-beta1", "acmecorp/scarlet.devil",
			"-p", tempDir.Path(), "-m", "Remilia Scarlet:remilia@acmecorp.cool").Assert(t, icmd.Success)
		metadata := golden.Get(t, tempDir.Join("scarlet.devil.dockerapp", "metadata.yml"))
		assert.Assert(t, golden.Bytes(metadata, "expected-fork-metadata.golden"))

		icmd.RunCommand(dockerApp, "fork", registry+"/acmecorp/simple.dockerapp:1.1.0-beta1",
			"-p", tempDir.Path(), "-m", "Remilia Scarlet:remilia@acmecorp.cool").Assert(t, icmd.Success)
		metadata2 := golden.Get(t, tempDir.Join("simple.dockerapp", "metadata.yml"))
		assert.Assert(t, golden.Bytes(metadata2, "expected-fork-metadata-no-rename.golden"))
	}
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

	// Test forking with external files
	tempDir := fs.NewDir(t, "dockerapptest")
	defer tempDir.Remove()

	icmd.RunCommand(dockerApp, "fork", registry+"/acmecorp/attachments.dockerapp:0.1.0",
		"-p", tempDir.Path()).Assert(t, icmd.Success)
	externalFile := golden.Get(t, tempDir.Join("attachments.dockerapp", "config.cfg"))
	assert.Assert(t, golden.Bytes(externalFile, filepath.Join("attachments.dockerapp", "config.cfg")))

	nestedAttachment := golden.Get(t, tempDir.Join("attachments.dockerapp", "nesteddir", "config2.cfg"))
	assert.Assert(t, golden.Bytes(nestedAttachment, filepath.Join("attachments.dockerapp", "nesteddir", "config2.cfg")))
}
