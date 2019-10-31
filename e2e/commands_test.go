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

		// Build the App
		cmd.Command = dockerCli.Command("app", "build", ".", "--file", filepath.Join(appPath, "my.dockerapp"), "--tag", "a-simple-tag", "--no-resolve-image")
		icmd.RunCmd(cmd).Assert(t, icmd.Success)

		// Render the App
		envParameters := map[string]string{}
		data, err := ioutil.ReadFile(filepath.Join(appPath, "env.yml"))
		assert.NilError(t, err)
		assert.NilError(t, yaml.Unmarshal(data, &envParameters))
		args := dockerCli.Command("app", "image", "render", "a-simple-tag", "--parameters-file", filepath.Join(appPath, "parameters-0.yml"))
		for k, v := range envParameters {
			args = append(args, "--set", fmt.Sprintf("%s=%s", k, v))
		}
		cmd.Command = args
		cmd.Env = append(cmd.Env, env...)
		t.Run("stdout", func(t *testing.T) {
			result := icmd.RunCmd(cmd).Assert(t, icmd.Success)
			expected := readFile(t, filepath.Join(appPath, "expected.txt"))
			actual := result.Stdout()
			assert.Assert(t, is.Equal(expected, actual), "rendering mismatch")
		})
		t.Run("file", func(t *testing.T) {
			cmd.Command = append(cmd.Command, "--output="+dir.Join("actual.yaml"))
			icmd.RunCmd(cmd).Assert(t, icmd.Success)
			expected := readFile(t, filepath.Join(appPath, "expected.txt"))
			actual := readFile(t, dir.Join("actual.yaml"))
			assert.Assert(t, is.Equal(expected, actual), "rendering mismatch")
		})
	}
}

func TestRenderAppNotFound(t *testing.T) {
	cmd, cleanup := dockerCli.createTestCmd()
	defer cleanup()

	appName := "non_existing_app:some_tag"
	cmd.Command = dockerCli.Command("app", "image", "render", appName)
	checkContains(t, icmd.RunCmd(cmd).Assert(t, icmd.Expected{ExitCode: 1}).Combined(),
		[]string{fmt.Sprintf("could not render %q: no such App image", appName)})
}

func TestRenderFormatters(t *testing.T) {
	runWithDindSwarmAndRegistry(t, func(info dindSwarmAndRegistryInfo) {
		cmd := info.configuredCmd

		contextPath := filepath.Join("testdata", "simple")
		cmd.Command = dockerCli.Command("app", "build", "--tag", "a-simple-tag", "--no-resolve-image", contextPath)
		icmd.RunCmd(cmd).Assert(t, icmd.Success)

		cmd.Command = dockerCli.Command("app", "image", "render", "--formatter", "json", "a-simple-tag")
		result := icmd.RunCmd(cmd).Assert(t, icmd.Success)
		golden.Assert(t, result.Stdout(), "expected-json-render.golden")

		cmd.Command = dockerCli.Command("app", "image", "render", "--formatter", "yaml", "a-simple-tag")
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
	runWithDindSwarmAndRegistry(t, func(info dindSwarmAndRegistryInfo) {
		cmd := info.configuredCmd

		// cwd = e2e
		dir := fs.NewDir(t, "detect-app-binary",
			fs.WithDir("attachments.dockerapp", fs.FromDir("testdata/attachments.dockerapp")))
		defer dir.Remove()

		cmd.Command = dockerCli.Command("app", "image", "inspect")
		cmd.Dir = dir.Path()
		icmd.RunCmd(cmd).Assert(t, icmd.Expected{
			ExitCode: 1,
			Err:      `"docker app image inspect" requires exactly 1 argument.`,
		})

		contextPath := filepath.Join("testdata", "simple")
		cmd.Command = dockerCli.Command("app", "build", "--tag", "simple-app:1.0.0", "--no-resolve-image", contextPath)
		cmd.Dir = ""
		icmd.RunCmd(cmd).Assert(t, icmd.Success)

		cmd.Command = dockerCli.Command("app", "image", "inspect", "simple-app:1.0.0")
		cmd.Dir = dir.Path()
		output := icmd.RunCmd(cmd).Assert(t, icmd.Success).Combined()
		golden.Assert(t, output, "app-inspect.golden")
	})
}

func TestRunOnlyOne(t *testing.T) {
	cmd, cleanup := dockerCli.createTestCmd()
	defer cleanup()

	cmd.Command = dockerCli.Command("app", "run")
	icmd.RunCmd(cmd).Assert(t, icmd.Expected{
		ExitCode: 1,
		Err:      `"docker app run" requires exactly 1 argument.`,
	})

	cmd.Command = dockerCli.Command("app", "run", "--cnab-bundle-json", "bundle.json", "myapp")
	icmd.RunCmd(cmd).Assert(t, icmd.Expected{
		ExitCode: 1,
		Err:      `"docker app run" cannot run a bundle and an App image`,
	})
}

func TestRunWithLabels(t *testing.T) {
	runWithDindSwarmAndRegistry(t, func(info dindSwarmAndRegistryInfo) {
		cmd := info.configuredCmd

		contextPath := filepath.Join("testdata", "simple")
		cmd.Command = dockerCli.Command("app", "build", "--tag", "myapp", contextPath)
		icmd.RunCmd(cmd).Assert(t, icmd.Success)

		cmd.Command = dockerCli.Command("app", "run", "myapp", "--name", "myapp", "--label", "label.key=labelValue")
		icmd.RunCmd(cmd).Assert(t, icmd.Success)

		services := []string{
			"myapp_db", "myapp_web", "myapp_api",
		}
		for _, service := range services {
			cmd.Command = dockerCli.Command("inspect", service)
			icmd.RunCmd(cmd).Assert(t, icmd.Expected{
				ExitCode: 0,
				Out:      `"label.key": "labelValue"`,
			})
		}
	})
}

func TestDockerAppLifecycle(t *testing.T) {
	runWithDindSwarmAndRegistry(t, func(info dindSwarmAndRegistryInfo) {
		cmd := info.configuredCmd
		appName := strings.ToLower(strings.Replace(t.Name(), "/", "_", 1))
		tmpDir := fs.NewDir(t, appName)
		defer tmpDir.Remove()

		cmd.Command = dockerCli.Command("app", "build", "--tag", appName, "testdata/simple")
		icmd.RunCmd(cmd).Assert(t, icmd.Success)

		// Install an illformed Docker Application Package
		cmd.Command = dockerCli.Command("app", "run", appName, "--set", "web_port=-1", "--name", appName)
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
		cmd.Command = dockerCli.Command("app", "update", appName)
		icmd.RunCmd(cmd).Assert(t, icmd.Expected{
			ExitCode: 1,
			Err:      fmt.Sprintf("Running App %q cannot be updated, please use 'docker app run' instead", appName),
		})

		// Install a Docker Application Package with an existing failed installation is fine
		cmd.Command = dockerCli.Command("app", "run", appName, "--name", appName)
		checkContains(t, icmd.RunCmd(cmd).Assert(t, icmd.Success).Combined(),
			[]string{
				fmt.Sprintf("WARNING: installing over previously failed installation %q", appName),
				fmt.Sprintf("Creating network %s_back", appName),
				fmt.Sprintf("Creating network %s_front", appName),
				fmt.Sprintf("Creating service %s_db", appName),
				fmt.Sprintf("Creating service %s_api", appName),
				fmt.Sprintf("Creating service %s_web", appName),
			})
		assertAppLabels(t, &cmd, appName, "db")

		// List the installed application
		cmd.Command = dockerCli.Command("app", "ls")
		checkContains(t, icmd.RunCmd(cmd).Assert(t, icmd.Success).Combined(),
			[]string{
				`INSTALLATION\s+APPLICATION\s+LAST ACTION\s+RESULT\s+CREATED\s+MODIFIED\s+REFERENCE`,
				fmt.Sprintf(`%s\s+simple \(1.1.0-beta1\)\s+install\s+success\s+.+second[s]?\sago\s+.+second[s]?\sago\s+`, appName),
			})

		// Installing again the same application is forbidden
		cmd.Command = dockerCli.Command("app", "run", appName, "--name", appName)
		icmd.RunCmd(cmd).Assert(t, icmd.Expected{
			ExitCode: 1,
			Err:      fmt.Sprintf("Installation %q already exists, use 'docker app update' instead", appName),
		})

		// Update the application, changing the port
		cmd.Command = dockerCli.Command("app", "update", appName, "--set", "web_port=8081")
		checkContains(t, icmd.RunCmd(cmd).Assert(t, icmd.Success).Combined(),
			[]string{
				fmt.Sprintf("Updating service %s_db", appName),
				fmt.Sprintf("Updating service %s_api", appName),
				fmt.Sprintf("Updating service %s_web", appName),
			})
		assertAppLabels(t, &cmd, appName, "db")

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
			"app", "run",
			"--credential", "secret1=foo",
			// secret2 deliberately omitted.
			"--credential", "secret3=baz",
			"--name", "missing",
			"--cnab-bundle-json", bundle,
		)
		result := icmd.RunCmd(cmd).Assert(t, icmd.Expected{
			ExitCode: 1,
			Out:      icmd.None,
		})
		golden.Assert(t, result.Stderr(), "credential-install-missing.golden")
	})

	t.Run("full", func(t *testing.T) {
		cmd.Command = dockerCli.Command(
			"app", "run",
			"--credential", "secret1=foo",
			"--credential", "secret2=bar",
			"--credential", "secret3=baz",
			"--name", "full",
			"--cnab-bundle-json", bundle,
		)
		result := icmd.RunCmd(cmd).Assert(t, icmd.Success)
		golden.Assert(t, result.Stdout(), "credential-install-full.golden")
	})

	t.Run("mixed-credstore", func(t *testing.T) {
		cmd.Command = dockerCli.Command(
			"app", "run",
			"--credential-set", "test-creds",
			"--credential", "secret3=xyzzy",
			"--name", "mixed-credstore",
			"--cnab-bundle-json", bundle,
		)
		result := icmd.RunCmd(cmd).Assert(t, icmd.Success)
		golden.Assert(t, result.Stdout(), "credential-install-mixed-credstore.golden")
	})

	t.Run("mixed-local-cred", func(t *testing.T) {
		cmd.Command = dockerCli.Command(
			"app", "run",
			"--credential-set", tmpDir.Join("local", "test-creds.yaml"),
			"--credential", "secret3=xyzzy",
			"--name", "mixed-local-cred",
			"--cnab-bundle-json", bundle,
		)
		result := icmd.RunCmd(cmd).Assert(t, icmd.Success)
		stdout := result.Stdout()
		golden.Assert(t, stdout, "credential-install-mixed-local-cred.golden")
	})

	t.Run("overload", func(t *testing.T) {
		cmd.Command = dockerCli.Command(
			"app", "run",
			"--credential-set", "test-creds",
			"--credential", "secret1=overload",
			"--credential", "secret3=xyzzy",
			"--name", "overload",
			"--cnab-bundle-json", bundle,
		)
		result := icmd.RunCmd(cmd).Assert(t, icmd.Expected{
			ExitCode: 1,
			Out:      icmd.None,
		})
		golden.Assert(t, result.Stderr(), "credential-install-overload.golden")
	})
}

func assertAppLabels(t *testing.T, cmd *icmd.Cmd, appName, containerName string) {
	cmd.Command = dockerCli.Command("inspect", fmt.Sprintf("%s_%s", appName, containerName))
	checkContains(t, icmd.RunCmd(*cmd).Assert(t, icmd.Success).Combined(),
		[]string{
			fmt.Sprintf(`"%s": "%s"`, internal.LabelAppNamespace, appName),
			fmt.Sprintf(`"%s": ".+"`, internal.LabelAppVersion),
		})
}

func checkContains(t *testing.T, combined string, expectedLines []string) {
	for _, expected := range expectedLines {
		exp := regexp.MustCompile(expected)
		assert.Assert(t, exp.MatchString(combined), "expected %q != actual %q", expected, combined)
	}
}
