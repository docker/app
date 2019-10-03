package e2e

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os/user"
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
)

func TestExitErrorCode(t *testing.T) {
	cmd, cleanup := dockerCli.createTestCmd()
	defer cleanup()

	cmd.Command = dockerCli.Command("app", "unknown_command")
	icmd.RunCmd(cmd).Assert(t, icmd.Expected{
		ExitCode: 1,
		Err:      "\"unknown_command\" is not a docker app command\nSee 'docker app --help'",
	})
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

func TestRenderFormatters(t *testing.T) {
	runWithDindSwarmAndRegistry(t, func(info dindSwarmAndRegistryInfo) {
		cmd := info.configuredCmd

		appPath := filepath.Join("testdata", "simple", "simple.dockerapp")
		cmd.Command = dockerCli.Command("app", "build", appPath)
		icmd.RunCmd(cmd).Assert(t, icmd.Success)

		cmd.Command = dockerCli.Command("app", "render", "--formatter", "json", appPath)
		result := icmd.RunCmd(cmd).Assert(t, icmd.Success)
		golden.Assert(t, result.Stdout(), "expected-json-render.golden")

		cmd.Command = dockerCli.Command("app", "render", "--formatter", "yaml", appPath)
		result = icmd.RunCmd(cmd).Assert(t, icmd.Success)
		golden.Assert(t, result.Stdout(), "expected-yaml-render.golden")
	})
}

func TestInit(t *testing.T) {
	cmd, cleanup := dockerCli.createTestCmd()
	defer cleanup()

	userData, _ := user.Current()
	currentUser := ""
	if userData != nil {
		currentUser = userData.Username
	}

	composeData := `version: "3.2"
services:
  nginx:
    image: nginx:latest
    command: nginx $NGINX_ARGS ${NGINX_DRY_RUN}
`
	meta := fmt.Sprintf(`# Version of the application
version: 0.1.0
# Name of the application
name: app-test
# A short description of the application
description: 
# List of application maintainers with name and email for each
maintainers:
  - name: %s
    email: 
`, currentUser)

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
		"--compose-file", tmpDir.Join(internal.ComposeFileName))
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
}

func TestInspectApp(t *testing.T) {
	cmd, cleanup := dockerCli.createTestCmd()
	defer cleanup()

	// cwd = e2e
	dir := fs.NewDir(t, "detect-app-binary",
		fs.WithDir("attachments.dockerapp", fs.FromDir("testdata/attachments.dockerapp")))
	defer dir.Remove()

	tmpDir := fs.NewDir(t, t.Name())
	defer tmpDir.Remove()

	cmd.Command = dockerCli.Command("app", "inspect")
	cmd.Dir = dir.Path()
	icmd.RunCmd(cmd).Assert(t, icmd.Expected{
		ExitCode: 1,
		Err:      "invalid reference format",
	})

	cmd.Command = dockerCli.Command("app", "bundle", filepath.Join("testdata", "simple", "simple.dockerapp"), "--output", tmpDir.Join("simple-bundle.json"), "--tag", "simple-app:1.0.0")
	cmd.Dir = ""
	icmd.RunCmd(cmd).Assert(t, icmd.Success)

	cmd.Command = dockerCli.Command("app", "inspect", "simple-app:1.0.0")
	cmd.Dir = dir.Path()
	output := icmd.RunCmd(cmd).Assert(t, icmd.Success).Combined()
	golden.Assert(t, output, "app-inspect.golden")
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
	cmd.Command = dockerCli.Command("app", "ls")
	checkContains(t, icmd.RunCmd(cmd).Assert(t, icmd.Success).Combined(),
		[]string{
			`INSTALLATION\s+APPLICATION\s+LAST ACTION\s+RESULT\s+CREATED\s+MODIFIED\s+REFERENCE`,
			fmt.Sprintf(`%s\s+simple \(1.1.0-beta1\)\s+install\s+failure\s+.+second[s]?\sago\s+.+second[s]?\sago\s+`, appName),
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

	// List the installed application
	cmd.Command = dockerCli.Command("app", "ls")
	checkContains(t, icmd.RunCmd(cmd).Assert(t, icmd.Success).Combined(),
		[]string{
			`INSTALLATION\s+APPLICATION\s+LAST ACTION\s+RESULT\s+CREATED\s+MODIFIED\s+REFERENCE`,
			fmt.Sprintf(`%s\s+simple \(1.1.0-beta1\)\s+install\s+success\s+.+second[s]?\sago\s+.+second[s]?\sago\s+`, appName),
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

	// Uninstall the application
	cmd.Command = dockerCli.Command("app", "rm", appName)
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
