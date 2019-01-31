package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gotest.tools/assert"
	"gotest.tools/icmd"
)

func TestExamplesAreValid(t *testing.T) {
	err := filepath.Walk("../examples", func(p string, info os.FileInfo, err error) error {
		appPath := filepath.Join(p, filepath.Base(p)+".dockerapp")
		_, statErr := os.Stat(appPath)
		switch {
		case strings.HasSuffix(p, "examples"):
			return nil
		case strings.HasSuffix(p, ".resources"):
			return filepath.SkipDir
		case !info.IsDir():
			return nil
		case os.IsNotExist(statErr):
			return nil
		default:
			result := icmd.RunCommand(dockerApp, "validate", appPath)
			result.Assert(t, icmd.Success)
			return filepath.SkipDir
		}
	})
	assert.NilError(t, err)
}
