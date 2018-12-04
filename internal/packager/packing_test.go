package packager

import (
	"bytes"
	"testing"

	"github.com/docker/app/types"
	"github.com/docker/docker/pkg/archive"
	"gotest.tools/assert"
	"gotest.tools/fs"
)

func TestPackInvocationImageContext(t *testing.T) {
	app, err := types.NewAppFromDefaultFiles("testdata/packages/packing.dockerapp")
	assert.NilError(t, err)
	buf := bytes.NewBuffer(nil)
	assert.NilError(t, PackInvocationImageContext(app, buf))
	dir := fs.NewDir(t, t.Name())
	defer dir.Remove()
	assert.NilError(t, archive.Untar(buf, dir.Path(), nil))
	expectedDir := fs.NewDir(t, t.Name(),
		fs.FromDir("testdata/packages"),
		fs.WithFile("Dockerfile", dockerFile),
		fs.WithFile(".dockerignore", dockerIgnore))
	defer expectedDir.Remove()
	assert.Assert(t, fs.Equal(dir.Path(), fs.ManifestFromDir(t, expectedDir.Path())))
}
