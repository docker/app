package e2e

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
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
		configDir := dockerCli.createTestConfig()
		defer os.RemoveAll(configDir)

		envParameters := map[string]string{}
		data, err := ioutil.ReadFile(filepath.Join(appPath, "env.yml"))
		assert.NilError(t, err)
		assert.NilError(t, yaml.Unmarshal(data, &envParameters))
		args := dockerCli.Command("app", "render", filepath.Join(appPath, "my.dockerapp"), "--parameters-files", filepath.Join(appPath, "parameters-0.yml"))
		for k, v := range envParameters {
			args = append(args, "--set", fmt.Sprintf("%s=%s", k, v))
		}
		result := icmd.RunCmd(icmd.Cmd{
			Command: args,
			Env:     append(os.Environ(), env...),
		}).Assert(t, icmd.Success)
		assert.Assert(t, is.Equal(readFile(t, filepath.Join(appPath, "expected.txt")), result.Stdout()), "rendering mismatch")
	}
}

func TestRenderFormatters(t *testing.T) {
	configDir := dockerCli.createTestConfig()
	defer os.RemoveAll(configDir)

	appPath := filepath.Join("testdata", "simple", "simple.dockerapp")
	cmd := icmd.Cmd{Command: dockerCli.Command("app", "render", "--formatter", "json", appPath)}
	result := icmd.RunCmd(cmd).Assert(t, icmd.Success)
	golden.Assert(t, result.Stdout(), "expected-json-render.golden")

	cmd.Command = dockerCli.Command("app", "render", "--formatter", "yaml", appPath)
	result = icmd.RunCmd(cmd).Assert(t, icmd.Success)
	golden.Assert(t, result.Stdout(), "expected-yaml-render.golden")
}

func TestInit(t *testing.T) {
	configDir := dockerCli.createTestConfig()
	defer os.RemoveAll(configDir)

	composeData := `version: "3.2"
services:
  nginx:
    image: nginx:latest
    command: nginx $NGINX_ARGS ${NGINX_DRY_RUN}
`
	meta := `# Version of the application
version: 0.1.0
# Name of the application
name: app-test
# A short description of the application
description: my cool app
# List of application maintainers with name and email for each
maintainers:
  - name: bob
    email: 
  - name: joe
    email: joe@joe.com
`
	envData := "# some comment\nNGINX_DRY_RUN=-t"
	tmpDir := fs.NewDir(t, "app_input",
		fs.WithFile(internal.ComposeFileName, composeData),
		fs.WithFile(".env", envData),
	)
	defer tmpDir.Remove()

	testAppName := "app-test"
	dirName := internal.DirNameFromAppName(testAppName)

	cmd := icmd.Cmd{Dir: tmpDir.Path(), Env: os.Environ()}

	cmd.Command = dockerCli.Command("app",
		"init", testAppName,
		"--compose-file", tmpDir.Join(internal.ComposeFileName),
		"--description", "my cool app",
		"--maintainer", "bob",
		"--maintainer", "joe:joe@joe.com")
	icmd.RunCmd(cmd).Assert(t, icmd.Success)

	manifest := fs.Expected(
		t,
		fs.WithMode(0755),
		fs.WithFile(internal.MetadataFileName, meta, fs.WithMode(0644)), // too many variables, cheating
		fs.WithFile(internal.ComposeFileName, composeData, fs.WithMode(0644)),
		fs.WithFile(internal.ParametersFileName, "NGINX_ARGS: FILL ME\nNGINX_DRY_RUN: -t\n", fs.WithMode(0644)),
	)
	assert.Assert(t, fs.Equal(tmpDir.Join(dirName), manifest))

	// validate metadata with JSON Schema
	cmd.Command = dockerCli.Command("app", "validate", testAppName)
	icmd.RunCmd(cmd).Assert(t, icmd.Success)

	// test single-file init
	cmd.Command = dockerCli.Command("app",
		"init", "tac",
		"--compose-file", tmpDir.Join(internal.ComposeFileName),
		"--description", "my cool app",
		"--maintainer", "bob",
		"--maintainer", "joe:joe@joe.com",
		"--single-file",
	)
	icmd.RunCmd(cmd).Assert(t, icmd.Success)

	appData, err := ioutil.ReadFile(tmpDir.Join("tac.dockerapp"))
	assert.NilError(t, err)
	golden.Assert(t, string(appData), "init-singlefile.dockerapp")
	// Check various commands work on single-file app package
	cmd.Command = dockerCli.Command("app", "inspect", "tac")
	icmd.RunCmd(cmd).Assert(t, icmd.Success)

	cmd.Command = dockerCli.Command("app", "render", "tac")
	icmd.RunCmd(cmd).Assert(t, icmd.Success)
}

func TestDetectApp(t *testing.T) {
	configDir := dockerCli.createTestConfig()
	defer os.RemoveAll(configDir)

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
		Command: dockerCli.Command("app", "inspect"),
		Dir:     dir.Path(),
	}).Assert(t, icmd.Success)
	icmd.RunCmd(icmd.Cmd{
		Command: dockerCli.Command("app", "inspect"),
		Dir:     dir.Join("attachments.dockerapp"),
	}).Assert(t, icmd.Success)
	icmd.RunCmd(icmd.Cmd{
		Command: dockerCli.Command("app", "inspect", "."),
		Dir:     dir.Join("attachments.dockerapp"),
	}).Assert(t, icmd.Success)
	result := icmd.RunCmd(icmd.Cmd{
		Command: dockerCli.Command("app", "inspect"),
		Dir:     dir.Join("render"),
	})
	result.Assert(t, icmd.Expected{
		ExitCode: 1,
		Err:      "Error: multiple applications found in current directory, specify the application name on the command line",
	})
}

func TestSplitMerge(t *testing.T) {
	configDir := dockerCli.createTestConfig()
	defer os.RemoveAll(configDir)

	tmpDir := fs.NewDir(t, "split_merge")
	defer tmpDir.Remove()

	cmd := icmd.Cmd{Command: dockerCli.Command("app", "merge", "testdata/render/envvariables/my.dockerapp", "--output", tmpDir.Join("remerged.dockerapp"))}
	icmd.RunCmd(cmd).Assert(t, icmd.Success)

	cmd.Dir = tmpDir.Path()

	// test that inspect works on single-file
	cmd.Command = dockerCli.Command("app", "inspect", "remerged")
	result := icmd.RunCmd(cmd).Assert(t, icmd.Success)
	golden.Assert(t, result.Combined(), "envvariables-inspect.golden")

	// split it
	cmd.Command = dockerCli.Command("app", "split", "remerged", "--output", "split.dockerapp")
	icmd.RunCmd(cmd).Assert(t, icmd.Success)

	cmd.Command = dockerCli.Command("app", "inspect", "remerged")
	result = icmd.RunCmd(cmd).Assert(t, icmd.Success)
	golden.Assert(t, result.Combined(), "envvariables-inspect.golden")

	// test inplace
	cmd.Command = dockerCli.Command("app", "merge", "split")
	icmd.RunCmd(cmd).Assert(t, icmd.Success)

	cmd.Command = dockerCli.Command("app", "split", "split")
	icmd.RunCmd(cmd).Assert(t, icmd.Success)
}

func TestBundle(t *testing.T) {
	configDir := dockerCli.createTestConfig()
	defer os.RemoveAll(configDir)

	tmpDir := fs.NewDir(t, t.Name())
	defer tmpDir.Remove()
	// Using a custom DOCKER_CONFIG to store contexts in a temporary directory
	cmd := icmd.Cmd{Env: os.Environ()}

	// Running a docker in docker to bundle the application
	dind := NewContainer("docker:18.09-dind", 2375)
	dind.Start(t)
	defer dind.Stop(t)

	// Create a build context
	cmd.Command = dockerCli.Command("context", "create", "build-context", "--docker", fmt.Sprintf(`"host=tcp://%s"`, dind.GetAddress(t)))
	icmd.RunCmd(cmd).Assert(t, icmd.Success)

	// The dind doesn't have the cnab-app-base image so we save it in order to load it later
	cmd.Command = dockerCli.Command("save", fmt.Sprintf("docker/cnab-app-base:%s", internal.Version), "--output", tmpDir.Join("cnab-app-base.tar.gz"))
	icmd.RunCmd(cmd).Assert(t, icmd.Success)

	cmd.Env = append(cmd.Env, "DOCKER_CONTEXT=build-context")
	cmd.Command = dockerCli.Command("load", "-i", tmpDir.Join("cnab-app-base.tar.gz"))
	icmd.RunCmd(cmd).Assert(t, icmd.Success)

	// Bundle the docker application package to a CNAB bundle, using the build-context.
	cmd.Command = dockerCli.Command("app", "bundle", filepath.Join("testdata", "simple", "simple.dockerapp"), "--out", tmpDir.Join("bundle.json"))
	icmd.RunCmd(cmd).Assert(t, icmd.Success)

	// Check the resulting CNAB bundle.json
	golden.Assert(t, string(golden.Get(t, tmpDir.Join("bundle.json"))), "simple-bundle.json.golden")

	// List the images on the build context daemon and checks the invocation image is there
	cmd.Command = dockerCli.Command("image", "ls", "--format", "{{.Repository}}:{{.Tag}}")
	icmd.RunCmd(cmd).Assert(t, icmd.Expected{ExitCode: 0, Out: "simple:1.1.0-beta1-invoc"})

	// Copy all the files from the invocation image and check them
	cmd.Command = dockerCli.Command("create", "--name", "invocation", "simple:1.1.0-beta1-invoc")
	id := strings.TrimSpace(icmd.RunCmd(cmd).Assert(t, icmd.Success).Stdout())
	cmd.Command = dockerCli.Command("cp", "invocation:/cnab/app/simple.dockerapp", tmpDir.Join("simple.dockerapp"))
	icmd.RunCmd(cmd).Assert(t, icmd.Success)
	cmd.Command = dockerCli.Command("rm", "--force", id)
	icmd.RunCmd(cmd).Assert(t, icmd.Success)

	appDir := filepath.Join("testdata", "simple", "simple.dockerapp")
	manifest := fs.Expected(
		t,
		fs.WithMode(0755),
		fs.WithFile(internal.MetadataFileName, readFile(t, filepath.Join(appDir, internal.MetadataFileName)), fs.WithMode(0644)),
		fs.WithFile(internal.ComposeFileName, readFile(t, filepath.Join(appDir, internal.ComposeFileName)), fs.WithMode(0644)),
		fs.WithFile(internal.ParametersFileName, readFile(t, filepath.Join(appDir, internal.ParametersFileName)), fs.WithMode(0644)),
	)

	assert.Assert(t, fs.Equal(tmpDir.Join("simple.dockerapp"), manifest))
}

func TestDockerAppLifecycle(t *testing.T) {
	configDir := dockerCli.createTestConfig()
	defer os.RemoveAll(configDir)

	tmpDir := fs.NewDir(t, t.Name())
	defer tmpDir.Remove()

	cmd := icmd.Cmd{
		Env: append(os.Environ(),
			fmt.Sprintf("DUFFLE_HOME=%s", tmpDir.Path()),
			"DOCKER_TARGET_CONTEXT=swarm-target-context",
		),
	}

	// Running a swarm using docker in docker to install the application
	// and run the invocation image
	swarm := NewContainer("docker:18.09-dind", 2375)
	swarm.Start(t)
	defer swarm.Stop(t)

	// The dind doesn't have the cnab-app-base image so we save it in order to load it later
	icmd.RunCommand(dockerCli.path, "save", fmt.Sprintf("docker/cnab-app-base:%s", internal.Version), "--output", tmpDir.Join("cnab-app-base.tar.gz")).Assert(t, icmd.Success)

	// We  need two contexts:
	// - one for `docker` so that it connects to the dind swarm created before
	// - the target context for the invocation image to install within the swarm
	cmd.Command = dockerCli.Command("context", "create", "swarm-context", "--docker", fmt.Sprintf(`"host=tcp://%s"`, swarm.GetAddress(t)), "--default-stack-orchestrator", "swarm")
	icmd.RunCmd(cmd).Assert(t, icmd.Success)

	// When creating a context on a Windows host we cannot use
	// the unix socket but it's needed inside the invocation image.
	// The workaround is to create a context with an empty host.
	// This host will default to the unix socket inside the
	// invocation image
	cmd.Command = dockerCli.Command("context", "create", "swarm-target-context", "--docker", "host=", "--default-stack-orchestrator", "swarm")
	icmd.RunCmd(cmd).Assert(t, icmd.Success)

	// Initialize the swarm
	cmd.Env = append(cmd.Env, "DOCKER_CONTEXT=swarm-context")
	cmd.Command = dockerCli.Command("swarm", "init")
	icmd.RunCmd(cmd).Assert(t, icmd.Success)

	// Load the needed base cnab image into the swarm docker engine
	cmd.Command = dockerCli.Command("load", "--input", tmpDir.Join("cnab-app-base.tar.gz"))
	icmd.RunCmd(cmd).Assert(t, icmd.Success)

	// Install a Docker Application Package
	cmd.Command = dockerCli.Command("app", "install", "testdata/simple/simple.dockerapp", "--name", t.Name())
	checkContains(t, icmd.RunCmd(cmd).Assert(t, icmd.Success).Combined(),
		[]string{
			fmt.Sprintf("Creating network %s_back", t.Name()),
			fmt.Sprintf("Creating network %s_front", t.Name()),
			fmt.Sprintf("Creating service %s_db", t.Name()),
			fmt.Sprintf("Creating service %s_api", t.Name()),
			fmt.Sprintf("Creating service %s_web", t.Name()),
		})

	// Query the application status
	cmd.Command = dockerCli.Command("app", "status", t.Name())
	checkContains(t, icmd.RunCmd(cmd).Assert(t, icmd.Success).Combined(),
		[]string{
			fmt.Sprintf("[[:alnum:]]+        %s_db    replicated          [0-1]/1                 postgres:9.3", t.Name()),
			fmt.Sprintf(`[[:alnum:]]+        %s_web   replicated          [0-1]/1                 nginx:latest        \*:8082->80/tcp`, t.Name()),
			fmt.Sprintf("[[:alnum:]]+        %s_api   replicated          [0-1]/1                 python:3.6", t.Name()),
		})

	// Upgrade the application, changing the port
	cmd.Command = dockerCli.Command("app", "upgrade", t.Name(), "--set", "web_port=8081")
	checkContains(t, icmd.RunCmd(cmd).Assert(t, icmd.Success).Combined(),
		[]string{
			fmt.Sprintf("Updating service %s_db", t.Name()),
			fmt.Sprintf("Updating service %s_api", t.Name()),
			fmt.Sprintf("Updating service %s_web", t.Name()),
		})

	// Query the application status again, the port should have change
	cmd.Command = dockerCli.Command("app", "status", t.Name())
	icmd.RunCmd(cmd).Assert(t, icmd.Expected{ExitCode: 0, Out: "8081"})

	// Uninstall the application
	cmd.Command = dockerCli.Command("app", "uninstall", t.Name())
	checkContains(t, icmd.RunCmd(cmd).Assert(t, icmd.Success).Combined(),
		[]string{
			fmt.Sprintf("Removing service %s_api", t.Name()),
			fmt.Sprintf("Removing service %s_db", t.Name()),
			fmt.Sprintf("Removing service %s_web", t.Name()),
			fmt.Sprintf("Removing network %s_front", t.Name()),
			fmt.Sprintf("Removing network %s_back", t.Name()),
		})
}

func checkContains(t *testing.T, combined string, expectedLines []string) {
	for _, expected := range expectedLines {
		exp := regexp.MustCompile(expected)
		assert.Assert(t, exp.MatchString(combined), expected, combined)
	}
}
