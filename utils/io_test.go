package utils

import (
	"testing"

	"gotest.tools/assert"
	"gotest.tools/fs"

	"strings"
)

var dummyData = []byte("hello world\n")

func TestCreateFileWithData(t *testing.T) {
	tmpdir := fs.NewDir(t, "iotest")
	defer tmpdir.Remove()
	path := tmpdir.Join("file.txt")

	err := CreateFileWithData(path, dummyData)
	assert.NilError(t, err, "failed to write data to file")

	manifest := fs.Expected(t,
		fs.WithFile("file.txt", string(dummyData),
			fs.WithMode(0644),
		),
	)
	assert.Assert(t, fs.Equal(tmpdir.Path(), manifest))
}

func TestCreateFileWithDataOverride(t *testing.T) {
	tmpdir := fs.NewDir(t, "iotest")
	defer tmpdir.Remove()
	path := tmpdir.Join("file.txt")

	err := CreateFileWithData(path, []byte("oops!"))
	assert.NilError(t, err, "failed to write data to file")
	err = CreateFileWithData(path, dummyData)
	assert.NilError(t, err, "failed to rewrite data to file")

	manifest := fs.Expected(t,
		fs.WithFile("file.txt", string(dummyData),
			fs.WithMode(0644),
		),
	)
	comp := fs.Equal(tmpdir.Path(), manifest)()
	assert.Assert(t, comp.Success())
}

func TestReadNewlineSeparatedList(t *testing.T) {
	reader := strings.NewReader("lorem\nipsum\r\ndolor sit\namet\n")
	results, err := ReadNewlineSeparatedList(reader)
	assert.NilError(t, err)
	expected := []string{
		"lorem", "ipsum", "dolor sit", "amet",
	}
	assert.DeepEqual(t, results, expected)
}

func TestReadNewlineSeparatedListWithEmptyLines(t *testing.T) {
	reader := strings.NewReader("\t\t\nlorem\n\nipsum\n   \ndolor sit\r\n\n\namet\n    ")
	results, err := ReadNewlineSeparatedList(reader)
	assert.NilError(t, err)
	expected := []string{
		"lorem", "ipsum", "dolor sit", "amet",
	}
	assert.DeepEqual(t, results, expected)
}

func TestReadNewlineSeparatedListSanitize(t *testing.T) {
	reader := strings.NewReader("\t\tlorem\nipsum\n\t\t\ndolor sit     \n \t amet \t\r\r\n")
	results, err := ReadNewlineSeparatedList(reader)
	assert.NilError(t, err)
	expected := []string{
		"lorem", "ipsum", "dolor sit", "amet",
	}
	assert.DeepEqual(t, results, expected)
}
