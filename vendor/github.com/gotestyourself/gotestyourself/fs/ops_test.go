package fs_test

import (
	"testing"

	"github.com/gotestyourself/gotestyourself/assert"
	"github.com/gotestyourself/gotestyourself/fs"
)

func TestFromDir(t *testing.T) {
	dir := fs.NewDir(t, "test-from-dir", fs.FromDir("testdata/copy-test"))
	defer dir.Remove()

	expected := fs.Expected(t,
		fs.WithFile("1", "1\n"),
		fs.WithDir("a",
			fs.WithFile("1", "1\n"),
			fs.WithFile("2", "2\n"),
			fs.WithDir("b",
				fs.WithFile("1", "1\n"))))

	assert.Assert(t, fs.Equal(dir.Path(), expected))
}
