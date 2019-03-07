package e2e

import (
	"fmt"
	"os"
	"path"
	"runtime"
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
			expectedOutput: "Status failed: action not defined for bundle",
			cnab:           "cnab-without-status",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			configDir := dockerCli.createTestConfig()
			defer os.RemoveAll(configDir)
			tmpDir := fs.NewDir(t, t.Name())
			defer tmpDir.Remove()
			testDir := path.Join("testdata", testCase.cnab)
			cmd := icmd.Cmd{Env: append(os.Environ(), fmt.Sprintf("DUFFLE_HOME=%s", tmpDir.Path()))}

			// We need to explicitly set the SYSTEMROOT on windows
			// otherwise we get the error:
			// "panic: failed to read random bytes: CryptAcquireContext: Provider DLL failed to initialize correctly."
			// See: https://github.com/golang/go/issues/25210
			if runtime.GOOS == "windows" {
				cmd.Env = append(cmd.Env, `SYSTEMROOT=C:\WINDOWS`)
			}
			// Build CNAB invocation image
			cmd.Command = dockerCli.Command("build", "--file", path.Join(testDir, "cnab", "build", "Dockerfile"), "--tag", fmt.Sprintf("e2e/%s:v0.1.0", testCase.cnab), testDir)
			icmd.RunCmd(cmd).Assert(t, icmd.Success)

			// docker app install
			cmd.Command = dockerCli.Command("app", "install", path.Join(testDir, "bundle.json"), "--name", testCase.name)
			icmd.RunCmd(cmd).Assert(t, icmd.Success)

			// docker app uninstall
			defer func() {
				cmd.Command = dockerCli.Command("app", "uninstall", testCase.name)
				icmd.RunCmd(cmd).Assert(t, icmd.Success)
			}()

			// docker app status
			cmd.Command = dockerCli.Command("app", "status", testCase.name)
			result := icmd.RunCmd(cmd)
			result.Assert(t, icmd.Expected{ExitCode: testCase.exitCode})
			assert.Assert(t, is.Contains(result.Combined(), testCase.expectedOutput))
		})
	}
}
