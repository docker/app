package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/deislabs/cnab-go/credentials"
	"github.com/docker/app/internal"
	"github.com/docker/app/internal/yaml"
	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
	"gotest.tools/fs"
	"gotest.tools/golden"
	"gotest.tools/icmd"
	"gotest.tools/skip"
)

// Skipping experimental e2e test on rendering with templates, as it needs now to set
// the DOCKERAPP_RENDERERS environment variable inside the invocation image,
// and the only way to do it is to add a new parameter to the bundle.json.
// This test should be unskipped or removed as soon as the templates are
// not experimental anymore, or are removed.
func TestRenderTemplates(t *testing.T) {
	t.Skip("renderer templates tests")
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
		cmd, cleanup := dockerCli.createTestCmd()
		defer cleanup()
		dir := fs.NewDir(t, "")
		defer dir.Remove()

		envParameters := map[string]string{}
		data, err := ioutil.ReadFile(filepath.Join(appPath, "env.yml"))
		assert.NilError(t, err)
		assert.NilError(t, yaml.Unmarshal(data, &envParameters))
		args := dockerCli.Command("app", "render", filepath.Join(appPath, "my.dockerapp"), "--parameters-file", filepath.Join(appPath, "parameters-0.yml"))
		for k, v := range envParameters {
			args = append(args, "--set", fmt.Sprintf("%s=%s", k, v))
		}
		cmd.Command = args
		cmd.Env = append(cmd.Env, env...)
		t.Run("stdout", func(t *testing.T) {
			result := icmd.RunCmd(cmd).Assert(t, icmd.Success)
			assert.Assert(t, is.Equal(readFile(t, filepath.Join(appPath, "expected.txt")), result.Stdout()), "rendering mismatch")
		})
		t.Run("file", func(t *testing.T) {
			cmd.Command = append(cmd.Command, "--output="+dir.Join("actual.yaml"))
			icmd.RunCmd(cmd).Assert(t, icmd.Success)
			assert.Assert(t, is.Equal(readFile(t, filepath.Join(appPath, "expected.txt")), readFile(t, dir.Join("actual.yaml"))), "rendering mismatch")
		})
	}
}

func TestRenderParametersWithSpecialChars(t *testing.T) {
	cmd, cleanup := dockerCli.createTestCmd()
	defer cleanup()
	dir := fs.NewDir(t, "")
	defer dir.Remove()

	appPath := filepath.Join("testdata", "parameters-with-special-chars")
	cmd.Command = dockerCli.Command("app", "render", filepath.Join(appPath, "my.dockerapp"))
	result := icmd.RunCmd(cmd).Assert(t, icmd.Success)
	golden.Assert(t, result.Stdout(), filepath.Join("parameters-with-special-chars", "expected.golden"))
}
func TestRenderFormatters(t *testing.T) {
	cmd, cleanup := dockerCli.createTestCmd()
	defer cleanup()

	appPath := filepath.Join("testdata", "simple", "simple.dockerapp")
	cmd.Command = dockerCli.Command("app", "render", "--formatter", "json", appPath)
	result := icmd.RunCmd(cmd).Assert(t, icmd.Success)
	golden.Assert(t, result.Stdout(), "expected-json-render.golden")

	cmd.Command = dockerCli.Command("app", "render", "--formatter", "yaml", appPath)
	result = icmd.RunCmd(cmd).Assert(t, icmd.Success)
	golden.Assert(t, result.Stdout(), "expected-yaml-render.golden")
}

func TestInit(t *testing.T) {
	cmd, cleanup := dockerCli.createTestCmd()
	defer cleanup()

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
  - name: dev1
    email: 
  - name: dev2
    email: dev2@example.com
`
	envData := "# some comment\nNGINX_DRY_RUN=-t"
	tmpDir := fs.NewDir(t, "app_input",
		fs.WithFile(internal.ComposeFileName, composeData),
		fs.WithFile(".env", envData),
	)
	defer tmpDir.Remove()

	testAppName := "app-test"
	dirName := internal.DirNameFromAppName(testAppName)

	cmd.Dir = tmpDir.Path()
	cmd.Command = dockerCli.Command("app",
		"init", testAppName,
		"--compose-file", tmpDir.Join(internal.ComposeFileName),
		"--description", "my cool app",
		"--maintainer", "dev1",
		"--maintainer", "dev2:dev2@example.com")
	stdOut := icmd.RunCmd(cmd).Assert(t, icmd.Success).Combined()
	golden.Assert(t, stdOut, "init-output.golden")

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
	stdOut = icmd.RunCmd(cmd).Assert(t, icmd.Success).Combined()
	golden.Assert(t, stdOut, "validate-output.golden")

	// test single-file init
	cmd.Command = dockerCli.Command("app",
		"init", "myapp",
		"--compose-file", tmpDir.Join(internal.ComposeFileName),
		"--description", "some description",
		"--maintainer", "dev1",
		"--maintainer", "dev2:dev2@example.com",
		"--single-file",
	)
	icmd.RunCmd(cmd).Assert(t, icmd.Success)

	appData, err := ioutil.ReadFile(tmpDir.Join("myapp.dockerapp"))
	assert.NilError(t, err)
	golden.Assert(t, string(appData), "init-singlefile.dockerapp")
	// Check various commands work on single-file app package
	cmd.Command = dockerCli.Command("app", "inspect", "myapp")
	icmd.RunCmd(cmd).Assert(t, icmd.Success)

	cmd.Command = dockerCli.Command("app", "render", "myapp")
	icmd.RunCmd(cmd).Assert(t, icmd.Success)
}

func TestDetectApp(t *testing.T) {
	cmd, cleanup := dockerCli.createTestCmd()
	defer cleanup()

	// cwd = e2e
	dir := fs.NewDir(t, "detect-app-binary",
		fs.WithDir("attachments.dockerapp", fs.FromDir("testdata/attachments.dockerapp")),
		fs.WithDir("render",
			fs.WithDir("app1.dockerapp", fs.FromDir("testdata/render/envvariables/my.dockerapp")),
			fs.WithDir("app2.dockerapp", fs.FromDir("testdata/render/envvariables/my.dockerapp")),
		),
	)
	defer dir.Remove()

	cmd.Command = dockerCli.Command("app", "inspect")
	cmd.Dir = dir.Path()
	icmd.RunCmd(cmd).Assert(t, icmd.Success)

	cmd.Command = dockerCli.Command("app", "inspect")
	cmd.Dir = dir.Join("attachments.dockerapp")
	icmd.RunCmd(cmd).Assert(t, icmd.Success)

	cmd.Command = dockerCli.Command("app", "inspect", ".")
	cmd.Dir = dir.Join("attachments.dockerapp")
	icmd.RunCmd(cmd).Assert(t, icmd.Success)

	cmd.Command = dockerCli.Command("app", "inspect")
	cmd.Dir = dir.Join("render")
	icmd.RunCmd(cmd).Assert(t, icmd.Expected{
		ExitCode: 1,
		Err:      "Error: multiple applications found in current directory, specify the application name on the command line",
	})
}

func patchEndOfLine(t *testing.T, eol []byte, name, fromDir, toDir string) {
	buf := golden.Get(t, filepath.Join(fromDir, name))
	bytes.ReplaceAll(buf, []byte{'\n'}, eol)
	assert.NilError(t, ioutil.WriteFile(filepath.Join(toDir, name), buf, 0644))
}

func TestSplitMerge(t *testing.T) {
	testCases := []struct {
		name       string
		lineEnding []byte
	}{
		{
			name:       "without-crlf",
			lineEnding: []byte{'\n'},
		},
		{
			name:       "with-crlf",
			lineEnding: []byte{'\r', '\n'},
		},
	}
	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			cmd, cleanup := dockerCli.createTestCmd()
			defer cleanup()

			tmpDir := fs.NewDir(t, "split_merge", fs.WithDir("my.dockerapp"))
			defer tmpDir.Remove()

			inputDir := filepath.Join("render", "envvariables", "my.dockerapp")
			testDataDir := tmpDir.Join("my.dockerapp")

			// replace the line ending from the versioned test data files and put them to a temp dir
			for _, name := range internal.FileNames {
				patchEndOfLine(t, test.lineEnding, name, inputDir, testDataDir)
			}

			// Merge a multi-files app to a single-file app
			cmd.Command = dockerCli.Command("app", "merge", testDataDir, "--output", tmpDir.Join("remerged.dockerapp"))
			icmd.RunCmd(cmd).Assert(t, icmd.Success)

			cmd.Dir = tmpDir.Path()

			// test that inspect works on single-file
			cmd.Command = dockerCli.Command("app", "inspect", "remerged")
			result := icmd.RunCmd(cmd).Assert(t, icmd.Success)
			golden.Assert(t, result.Combined(), "envvariables-inspect.golden")

			// split it
			cmd.Command = dockerCli.Command("app", "split", "remerged", "--output", "split.dockerapp")
			icmd.RunCmd(cmd).Assert(t, icmd.Success)

			// test that inspect still works
			cmd.Command = dockerCli.Command("app", "inspect", "remerged")
			result = icmd.RunCmd(cmd).Assert(t, icmd.Success)
			golden.Assert(t, result.Combined(), "envvariables-inspect.golden")

			// the split app must be the same as the original one
			manifest := fs.Expected(
				t,
				fs.WithMode(0755),
				fs.WithFile(internal.MetadataFileName, string(golden.Get(t, path.Join(testDataDir, internal.MetadataFileName))), fs.WithMode(0644)),
				fs.WithFile(internal.ComposeFileName, string(golden.Get(t, path.Join(testDataDir, internal.ComposeFileName))), fs.WithMode(0644)),
				fs.WithFile(internal.ParametersFileName, string(golden.Get(t, path.Join(testDataDir, internal.ParametersFileName))), fs.WithMode(0644)),
			)
			assert.Assert(t, fs.Equal(tmpDir.Join("split.dockerapp"), manifest))

			// test inplace
			cmd.Command = dockerCli.Command("app", "merge", "split")
			icmd.RunCmd(cmd).Assert(t, icmd.Success)

			cmd.Command = dockerCli.Command("app", "split", "split")
			icmd.RunCmd(cmd).Assert(t, icmd.Success)
		})
	}
}

func TestBundle(t *testing.T) {
	cmd, cleanup := dockerCli.createTestCmd()
	defer cleanup()

	tmpDir := fs.NewDir(t, t.Name())
	defer tmpDir.Remove()

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

	testCases := []struct {
		name           string
		cmd            []string
		invocImage     string
		expectedBundle string
	}{
		{
			name:           "simple-bundle",
			cmd:            dockerCli.Command("app", "bundle", filepath.Join("testdata", "simple", "simple.dockerapp"), "--output", tmpDir.Join("simple-bundle.json")),
			invocImage:     "simple:1.1.0-beta1-invoc",
			expectedBundle: "simple-bundle.json.golden",
		},
		{
			name:           "bundle-with-tag",
			cmd:            dockerCli.Command("app", "bundle", filepath.Join("testdata", "simple", "simple.dockerapp"), "--tag", "myimage:mytag", "--output", tmpDir.Join("bundle-with-tag.json")),
			invocImage:     "myimage:mytag-invoc",
			expectedBundle: "bundle-with-tag.json.golden",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testDir := fs.NewDir(t, "")
			defer testDir.Remove()

			// Bundle the docker application package to a CNAB bundle, using the build-context.
			cmd.Command = tc.cmd
			icmd.RunCmd(cmd).Assert(t, icmd.Success)

			// Check the resulting CNAB bundle.json
			golden.Assert(t, string(golden.Get(t, tmpDir.Join(tc.name+".json"))), tc.expectedBundle)

			// List the images on the build context daemon and checks the invocation image is there
			cmd.Command = dockerCli.Command("image", "ls", "--format", "{{.Repository}}:{{.Tag}}")
			icmd.RunCmd(cmd).Assert(t, icmd.Expected{ExitCode: 0, Out: tc.invocImage})

			// Copy all the files from the invocation image and check them
			cmd.Command = dockerCli.Command("create", "--name", "invocation", tc.invocImage)
			id := strings.TrimSpace(icmd.RunCmd(cmd).Assert(t, icmd.Success).Stdout())
			cmd.Command = dockerCli.Command("cp", "invocation:/cnab/app/simple.dockerapp", testDir.Join("simple.dockerapp"))
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
			assert.Assert(t, fs.Equal(testDir.Join("simple.dockerapp"), manifest))
		})
	}
}

func TestDockerAppLifecycle(t *testing.T) {
	t.Run("withBindMounts", func(t *testing.T) {
		testDockerAppLifecycle(t, true)
	})
	t.Run("withoutBindMounts", func(t *testing.T) {
		testDockerAppLifecycle(t, false)
	})
}

func testDockerAppLifecycle(t *testing.T, useBindMount bool) {
	cmd, cleanup := dockerCli.createTestCmd()
	defer cleanup()
	appName := strings.Replace(t.Name(), "/", "_", 1)
	tmpDir := fs.NewDir(t, appName)
	defer tmpDir.Remove()
	// Running a swarm using docker in docker to install the application
	// and run the invocation image
	swarm := NewContainer("docker:18.09-dind", 2375)
	swarm.Start(t)
	defer swarm.Stop(t)
	initializeDockerAppEnvironment(t, &cmd, tmpDir, swarm, useBindMount)

	// Install an illformed Docker Application Package
	cmd.Command = dockerCli.Command("app", "install", "testdata/simple/simple.dockerapp", "--set", "web_port=-1", "--name", appName)
	icmd.RunCmd(cmd).Assert(t, icmd.Expected{
		ExitCode: 1,
		Err:      "error decoding 'Ports': Invalid hostPort: -1",
	})

	// List the installation and check the failed status
	cmd.Command = dockerCli.Command("app", "list")
	checkContains(t, icmd.RunCmd(cmd).Assert(t, icmd.Success).Combined(),
		[]string{
			`INSTALLATION\s+APPLICATION\s+LAST ACTION\s+RESULT\s+CREATED\s+MODIFIED\s+REFERENCE`,
			fmt.Sprintf(`%s\s+simple \(1.1.0-beta1\)\s+install\s+failure\s+.+second[s]?\s+.+second[s]?\s+`, appName),
		})

	// Upgrading a failed installation is not allowed
	cmd.Command = dockerCli.Command("app", "upgrade", appName)
	icmd.RunCmd(cmd).Assert(t, icmd.Expected{
		ExitCode: 1,
		Err:      fmt.Sprintf("Installation %q has failed and cannot be upgraded, reinstall it using 'docker app install'", appName),
	})

	// Install a Docker Application Package with an existing failed installation is fine
	cmd.Command = dockerCli.Command("app", "install", "testdata/simple/simple.dockerapp", "--name", appName)
	checkContains(t, icmd.RunCmd(cmd).Assert(t, icmd.Success).Combined(),
		[]string{
			fmt.Sprintf("WARNING: installing over previously failed installation %q", appName),
			fmt.Sprintf("Creating network %s_back", appName),
			fmt.Sprintf("Creating network %s_front", appName),
			fmt.Sprintf("Creating service %s_db", appName),
			fmt.Sprintf("Creating service %s_api", appName),
			fmt.Sprintf("Creating service %s_web", appName),
		})

	// Query the application status
	cmd.Command = dockerCli.Command("app", "status", appName)
	checkContains(t, icmd.RunCmd(cmd).Assert(t, icmd.Success).Combined(),
		[]string{
			`INSTALLATION
------------
Name:         TestDockerAppLifecycle_.*
Created:      .*
Modified:     .*
Revision:     .*
Last Action:  install
Result:       SUCCESS
Orchestrator: swarm

APPLICATION
-----------
Name:      simple
Version:   1.1.0-beta1
Reference:.*

PARAMETERS
----------
api_host:      example.com
static_subdir: data/static
web_port:      8082

STATUS
------`,
			fmt.Sprintf("[[:alnum:]]+        %s_db    replicated          [0-1]/1                 postgres:9.3", appName),
			fmt.Sprintf(`[[:alnum:]]+        %s_web   replicated          [0-1]/1                 nginx:latest        \*:8082->80/tcp`, appName),
			fmt.Sprintf("[[:alnum:]]+        %s_api   replicated          [0-1]/1                 python:3.6", appName),
		})

	// List the installed application
	cmd.Command = dockerCli.Command("app", "list")
	checkContains(t, icmd.RunCmd(cmd).Assert(t, icmd.Success).Combined(),
		[]string{
			`INSTALLATION\s+APPLICATION\s+LAST ACTION\s+RESULT\s+CREATED\s+MODIFIED\s+REFERENCE`,
			fmt.Sprintf(`%s\s+simple \(1.1.0-beta1\)\s+install\s+success\s+.+second[s]?\s+.+second[s]?\s+`, appName),
		})

	// Installing again the same application is forbidden
	cmd.Command = dockerCli.Command("app", "install", "testdata/simple/simple.dockerapp", "--name", appName)
	icmd.RunCmd(cmd).Assert(t, icmd.Expected{
		ExitCode: 1,
		Err:      fmt.Sprintf("Installation %q already exists, use 'docker app upgrade' instead", appName),
	})

	// Upgrade the application, changing the port
	cmd.Command = dockerCli.Command("app", "upgrade", appName, "--set", "web_port=8081")
	checkContains(t, icmd.RunCmd(cmd).Assert(t, icmd.Success).Combined(),
		[]string{
			fmt.Sprintf("Updating service %s_db", appName),
			fmt.Sprintf("Updating service %s_api", appName),
			fmt.Sprintf("Updating service %s_web", appName),
		})

	// Query the application status again, the port should have change
	cmd.Command = dockerCli.Command("app", "status", appName)
	icmd.RunCmd(cmd).Assert(t, icmd.Expected{ExitCode: 0, Out: "8081"})

	// Uninstall the application
	cmd.Command = dockerCli.Command("app", "uninstall", appName)
	checkContains(t, icmd.RunCmd(cmd).Assert(t, icmd.Success).Combined(),
		[]string{
			fmt.Sprintf("Removing service %s_api", appName),
			fmt.Sprintf("Removing service %s_db", appName),
			fmt.Sprintf("Removing service %s_web", appName),
			fmt.Sprintf("Removing network %s_front", appName),
			fmt.Sprintf("Removing network %s_back", appName),
		})
}

func TestCredentials(t *testing.T) {
	credSet := &credentials.CredentialSet{
		Name: "test-creds",
		Credentials: []credentials.CredentialStrategy{
			{
				Name: "secret1",
				Source: credentials.Source{
					Value: "secret1value",
				},
			},
			{
				Name: "secret2",
				Source: credentials.Source{
					Value: "secret2value",
				},
			},
		},
	}
	// Create a tmp dir with a credential store
	cmd, cleanup := dockerCli.createTestCmd(
		withCredentialSet(t, "default", credSet),
	)
	defer cleanup()
	// Create a local credentialSet

	buf, err := json.Marshal(credSet)
	assert.NilError(t, err)
	bundleJSON := golden.Get(t, "credential-install-bundle.json")
	tmpDir := fs.NewDir(t, t.Name(),
		fs.WithFile("bundle.json", "", fs.WithBytes(bundleJSON)),
		fs.WithDir("local",
			fs.WithFile("test-creds.yaml", "", fs.WithBytes(buf)),
		),
	)
	defer tmpDir.Remove()

	bundle := tmpDir.Join("bundle.json")

	t.Run("missing", func(t *testing.T) {
		cmd.Command = dockerCli.Command(
			"app", "install",
			"--credential", "secret1=foo",
			// secret2 deliberately omitted.
			"--credential", "secret3=baz",
			"--name", "missing", bundle,
		)
		result := icmd.RunCmd(cmd).Assert(t, icmd.Expected{
			ExitCode: 1,
			Out:      icmd.None,
		})
		golden.Assert(t, result.Stderr(), "credential-install-missing.golden")
	})

	t.Run("full", func(t *testing.T) {
		cmd.Command = dockerCli.Command(
			"app", "install",
			"--credential", "secret1=foo",
			"--credential", "secret2=bar",
			"--credential", "secret3=baz",
			"--name", "full", bundle,
		)
		result := icmd.RunCmd(cmd).Assert(t, icmd.Success)
		golden.Assert(t, result.Stdout(), "credential-install-full.golden")
	})

	t.Run("mixed-credstore", func(t *testing.T) {
		cmd.Command = dockerCli.Command(
			"app", "install",
			"--credential-set", "test-creds",
			"--credential", "secret3=xyzzy",
			"--name", "mixed-credstore", bundle,
		)
		result := icmd.RunCmd(cmd).Assert(t, icmd.Success)
		golden.Assert(t, result.Stdout(), "credential-install-mixed-credstore.golden")
	})

	t.Run("mixed-local-cred", func(t *testing.T) {
		cmd.Command = dockerCli.Command(
			"app", "install",
			"--credential-set", tmpDir.Join("local", "test-creds.yaml"),
			"--credential", "secret3=xyzzy",
			"--name", "mixed-local-cred", bundle,
		)
		result := icmd.RunCmd(cmd).Assert(t, icmd.Success)
		golden.Assert(t, result.Stdout(), "credential-install-mixed-local-cred.golden")
	})

	t.Run("overload", func(t *testing.T) {
		cmd.Command = dockerCli.Command(
			"app", "install",
			"--credential-set", "test-creds",
			"--credential", "secret1=overload",
			"--credential", "secret3=xyzzy",
			"--name", "overload", bundle,
		)
		result := icmd.RunCmd(cmd).Assert(t, icmd.Expected{
			ExitCode: 1,
			Out:      icmd.None,
		})
		golden.Assert(t, result.Stderr(), "credential-install-overload.golden")
	})
}

func initializeDockerAppEnvironment(t *testing.T, cmd *icmd.Cmd, tmpDir *fs.Dir, swarm *Container, useBindMount bool) {
	cmd.Env = append(cmd.Env, "DOCKER_TARGET_CONTEXT=swarm-target-context")

	// The dind doesn't have the cnab-app-base image so we save it in order to load it later
	icmd.RunCommand(dockerCli.path, "save", fmt.Sprintf("docker/cnab-app-base:%s", internal.Version), "--output", tmpDir.Join("cnab-app-base.tar.gz")).Assert(t, icmd.Success)

	// We  need two contexts:
	// - one for `docker` so that it connects to the dind swarm created before
	// - the target context for the invocation image to install within the swarm
	cmd.Command = dockerCli.Command("context", "create", "swarm-context", "--docker", fmt.Sprintf(`"host=tcp://%s"`, swarm.GetAddress(t)), "--default-stack-orchestrator", "swarm")
	icmd.RunCmd(*cmd).Assert(t, icmd.Success)

	// When creating a context on a Windows host we cannot use
	// the unix socket but it's needed inside the invocation image.
	// The workaround is to create a context with an empty host.
	// This host will default to the unix socket inside the
	// invocation image
	host := "host="
	if !useBindMount {
		host += fmt.Sprintf("tcp://%s", swarm.GetPrivateAddress(t))
	}

	cmd.Command = dockerCli.Command("context", "create", "swarm-target-context", "--docker", host, "--default-stack-orchestrator", "swarm")
	icmd.RunCmd(*cmd).Assert(t, icmd.Success)

	// Initialize the swarm
	cmd.Env = append(cmd.Env, "DOCKER_CONTEXT=swarm-context")
	cmd.Command = dockerCli.Command("swarm", "init")
	icmd.RunCmd(*cmd).Assert(t, icmd.Success)

	// Load the needed base cnab image into the swarm docker engine
	cmd.Command = dockerCli.Command("load", "--input", tmpDir.Join("cnab-app-base.tar.gz"))
	icmd.RunCmd(*cmd).Assert(t, icmd.Success)
}

func checkContains(t *testing.T, combined string, expectedLines []string) {
	for _, expected := range expectedLines {
		exp := regexp.MustCompile(expected)
		assert.Assert(t, exp.MatchString(combined), "expected %q != actual %q", expected, combined)
	}
}
