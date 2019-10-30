package e2e

import (
	"fmt"
	"path"
	"testing"

	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
	"gotest.tools/icmd"
)

func TestCallCustomStatusAction(t *testing.T) {
	testCases := []struct {
		name           string
		exitCode       int
		expectedOutput string
		cnab           string
	}{
		{
			name:           "validCustomDockerStatusAction",
			exitCode:       0,
			expectedOutput: "com.docker.app.status",
			cnab:           "cnab-with-docker-status",
		},
		{
			name:           "validCustomStandardStatusAction",
			exitCode:       0,
			expectedOutput: "io.cnab.status",
			cnab:           "cnab-with-standard-status",
		},
		// A CNAB bundle without standard or docker status action still can output
		// some informations about the installation.
		{
			name:           "missingCustomStatusAction",
			exitCode:       0,
			expectedOutput: "Name:        missingCustomStatusAction",
			cnab:           "cnab-without-status",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			cmd, cleanup := dockerCli.createTestCmd()
			defer cleanup()
			testDir := path.Join("testdata", testCase.cnab)

			// Build CNAB invocation image
			cmd.Command = dockerCli.Command("build", "--file", path.Join(testDir, "cnab", "build", "Dockerfile"), "--tag", fmt.Sprintf("e2e/%s:v0.1.0", testCase.cnab), testDir)
			icmd.RunCmd(cmd).Assert(t, icmd.Success)

			// docker app install
			cmd.Command = dockerCli.Command("app", "run", "--cnab-bundle-json", path.Join(testDir, "bundle.json"), "--name", testCase.name)
			icmd.RunCmd(cmd).Assert(t, icmd.Success)

			// docker app uninstall
			defer func() {
				cmd.Command = dockerCli.Command("app", "rm", testCase.name)
				icmd.RunCmd(cmd).Assert(t, icmd.Success)
			}()
		})
	}
}

func TestCnabParameters(t *testing.T) {
	cmd, cleanup := dockerCli.createTestCmd()
	defer cleanup()
	testDir := path.Join("testdata", "cnab-parameters")

	// Build CNAB invocation image
	cmd.Command = dockerCli.Command("build", "--file", path.Join(testDir, "cnab", "build", "Dockerfile"), "--tag", "e2e/cnab-parameters:v0.1.0", testDir)
	icmd.RunCmd(cmd).Assert(t, icmd.Success)

	// docker app uninstall
	defer func() {
		cmd.Command = dockerCli.Command("app", "rm", "cnab-parameters")
		icmd.RunCmd(cmd).Assert(t, icmd.Success)
	}()

	// docker app install
	cmd.Command = dockerCli.Command("app", "run", "--cnab-bundle-json", path.Join(testDir, "bundle.json"), "--name", "cnab-parameters",
		"--set", "boolParam=true",
		"--set", "stringParam=value",
		"--set", "intParam=42")
	result := icmd.RunCmd(cmd).Assert(t, icmd.Success)
	expectedOutput := `boolParam=true
stringParam=value
intParam=42`
	assert.Assert(t, is.Contains(result.Combined(), expectedOutput))
}
