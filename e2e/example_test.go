package e2e

import (
	"os"
	"path/filepath"
	"testing"

	"gotest.tools/assert"
	"gotest.tools/icmd"
)

func TestExamplesAreValid(t *testing.T) {
	cmd, cleanup := dockerCli.createTestCmd()
	defer cleanup()

	err := filepath.Walk("../examples", func(p string, info os.FileInfo, err error) error {
		if filepath.Ext(p) == ".dockerapp" {
			t.Log("Validate example: " + p)
			cmd.Command = dockerCli.Command("app", "validate", p)
			icmd.RunCmd(cmd).Assert(t, icmd.Success)
		}
		return nil
	})
	assert.NilError(t, err)
}
