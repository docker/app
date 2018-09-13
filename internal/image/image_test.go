package image

import (
	"testing"

	"gotest.tools/assert"
)

func TestChangeImageRepository(t *testing.T) {
	res, err := ChangeImageRepository("foo/bar:1.2", "localhost:5000")
	assert.NilError(t, err)
	assert.Equal(t, res, "localhost:5000/foo/bar:1.2")
	res, err = ChangeImageRepository("repo.io/foo/bar:1.2", "localhost:5000")
	assert.NilError(t, err)
	assert.Equal(t, res, "localhost:5000/foo/bar:1.2")
	res, err = ChangeImageRepository("repo.io:3000/foo/bar:1.2", "localhost:5000")
	assert.NilError(t, err)
	assert.Equal(t, res, "localhost:5000/foo/bar:1.2")
	res, err = ChangeImageRepository("foo/bar", "localhost:5000")
	assert.NilError(t, err)
	assert.Equal(t, res, "localhost:5000/foo/bar")
}
