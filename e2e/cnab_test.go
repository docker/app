package e2e

import (
	"fmt"
	"path"
	"testing"

	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
	"gotest.tools/fs"
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
			name:           "validCustomStatusAction",
			exitCode:       0,
			expectedOutput: "Status action",
			cnab:           "cnab-with-status",
		},
		{
			name:           "missingCustomStatusAction",
			exitCode:       1,
			expectedOutput: "Error: Status failed: action not defined for bundle",
			cnab:           "cnab-without-status",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			tmpDir := fs.NewDir(t, t.Name())
			defer tmpDir.Remove()
			testDir := path.Join("testdata", testCase.cnab)
			cmd := icmd.Cmd{
				Env: []string{fmt.Sprintf("DUFFLE_HOME=%s", tmpDir.Path())},
			}

			// Build CNAB invocation image
			cmd.Command = []string{"docker", "build", "-f", path.Join(testDir, "cnab", "build", "Dockerfile"), "-t", fmt.Sprintf("e2e/%s:v0.1.0", testCase.cnab), testDir}
			icmd.RunCmd(cmd).Assert(t, icmd.Success)

			// docker-app install
			cmd.Command = []string{dockerApp, "install", path.Join(testDir, "bundle.json"), "--name", testCase.name}
			icmd.RunCmd(cmd).Assert(t, icmd.Success)

			// docker-app uninstall
			defer func() {
				cmd.Command = []string{dockerApp, "uninstall", testCase.name}
				icmd.RunCmd(cmd).Assert(t, icmd.Success)
			}()

			// docker-app status
			cmd.Command = []string{dockerApp, "status", testCase.name}
			result := icmd.RunCmd(cmd)
			result.Assert(t, icmd.Expected{ExitCode: testCase.exitCode})
			assert.Assert(t, is.Contains(result.Combined(), testCase.expectedOutput))
		})
	}
}
