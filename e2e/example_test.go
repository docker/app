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
		switch {
		case strings.HasSuffix(p, "examples"):
			return nil
		case strings.HasSuffix(p, ".resources"):
			return filepath.SkipDir
		case !info.IsDir():
			return nil
		default:
			result := icmd.RunCommand(dockerApp, "app", "validate", filepath.Join(p, filepath.Base(p)+".dockerapp"))
			result.Assert(t, icmd.Success)
			return filepath.SkipDir
		}
	})
	assert.NilError(t, err)
}
