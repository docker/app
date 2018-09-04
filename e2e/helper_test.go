package e2e

import (
	"path/filepath"
	"testing"

	"gotest.tools/assert"
	"gotest.tools/fs"
	"gotest.tools/icmd"
)

func startRegistry(t *testing.T) *Container {
	c := &Container{image: "registry:2", privatePort: 5000}
	c.Start(t)
	return c
}

func runHelmCommand(t *testing.T, args ...string) *fs.Dir {
	t.Helper()
	abs, err := filepath.Abs(".")
	assert.NilError(t, err)
	dir := fs.NewDir(t, t.Name(), fs.FromDir(abs))
	result := icmd.RunCmd(icmd.Cmd{
		Command: append([]string{dockerApp}, args...),
		Dir:     dir.Path(),
	})
	result.Assert(t, icmd.Success)
	return dir
}
