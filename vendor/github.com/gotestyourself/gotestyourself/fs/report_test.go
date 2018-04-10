package fs

import (
	"fmt"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/gotestyourself/gotestyourself/assert"
	is "github.com/gotestyourself/gotestyourself/assert/cmp"
)

func TestEqualMissingRoot(t *testing.T) {
	result := Equal("/bogus/path/does/not/exist", Expected(t))()
	assert.Assert(t, !result.Success())
	expected := "stat /bogus/path/does/not/exist: no such file or directory"
	if runtime.GOOS == "windows" {
		expected = "CreateFile /bogus/path/does/not/exist"
	}
	assert.Assert(t, is.Contains(result.(cmpFailure).FailureMessage(), expected))
}

func TestEqualModeMismatch(t *testing.T) {
	dir := NewDir(t, t.Name(), WithMode(0500))
	defer dir.Remove()

	result := Equal(dir.Path(), Expected(t))()
	assert.Assert(t, !result.Success())
	expected := fmtExpected(`directory %s does not match expected:
/
  mode: expected drwx------ got dr-x------
`, dir.Path())
	if runtime.GOOS == "windows" {
		expected = fmtExpected(`directory %s does not match expected:
\
  mode: expected drwxrwxrwx got dr-xr-xr-x
`, dir.Path())
	}
	assert.Equal(t, result.(cmpFailure).FailureMessage(), expected)
}

func TestEqualRootIsAFile(t *testing.T) {
	file := NewFile(t, t.Name())
	defer file.Remove()

	result := Equal(file.Path(), Expected(t))()
	assert.Assert(t, !result.Success())
	expected := fmt.Sprintf("path %s must be a directory", file.Path())
	assert.Equal(t, result.(cmpFailure).FailureMessage(), expected)
}

func TestEqualSuccess(t *testing.T) {
	dir := NewDir(t, t.Name(), WithMode(0700))
	defer dir.Remove()

	assert.Assert(t, Equal(dir.Path(), Expected(t)))
}

func TestEqualDirectoryHasWithExtraFiles(t *testing.T) {
	dir := NewDir(t, t.Name(),
		WithFile("extra1", "content"))
	defer dir.Remove()

	manifest := Expected(t, WithFile("file1", "content"))
	result := Equal(dir.Path(), manifest)()
	assert.Assert(t, !result.Success())
	expected := fmtExpected(`directory %s does not match expected:
/
  file1: expected file to exist
  extra1: unexpected file
`, dir.Path())
	assert.Equal(t, result.(cmpFailure).FailureMessage(), expected)
}

func fmtExpected(format string, args ...interface{}) string {
	return filepath.FromSlash(fmt.Sprintf(format, args...))
}

func TestEqualWithAllowAnyFileContent(t *testing.T) {
	dir := NewDir(t, t.Name(),
		WithFile("data", "this is some data"))
	defer dir.Remove()

	expected := Expected(t,
		WithFile("data", "different content", MatchAnyFileContent))
	assert.Assert(t, Equal(dir.Path(), expected))
}

func TestEqualWithFileContent(t *testing.T) {
	dir := NewDir(t, "assert-test-root",
		WithFile("file1", "line1\nline2\nline3"))
	defer dir.Remove()

	manifest := Expected(t,
		WithFile("file1", "line2\nline3"))

	result := Equal(dir.Path(), manifest)()
	expected := fmtExpected(`directory %s does not match expected:
/file1
  content:
    --- expected
    +++ actual
    @@ -1,2 +1,3 @@
    +line1
     line2
     line3
`, dir.Path())
	assert.Equal(t, result.(cmpFailure).FailureMessage(), expected)
}

func TestEqualDirectoryWithAllowExtraFiles(t *testing.T) {
	file1 := WithFile("file1", "same in both")
	dir := NewDir(t, t.Name(),
		file1,
		WithFile("extra", "some content"))
	defer dir.Remove()

	expected := Expected(t, file1, MatchExtraFiles)
	assert.Assert(t, Equal(dir.Path(), expected))
}

func TestEqualManyFailures(t *testing.T) {
	dir := NewDir(t, t.Name(),
		WithFile("file1", "same in both"),
		WithFile("extra", "some content"),
		WithSymlink("sym1", "extra"))
	defer dir.Remove()

	manifest := Expected(t,
		WithDir("subdir",
			WithFile("somefile", "")),
		WithFile("file1", "not the\nsame in both"))

	result := Equal(dir.Path(), manifest)()
	assert.Assert(t, !result.Success())

	expected := fmtExpected(`directory %s does not match expected:
/
  subdir: expected directory to exist
  extra: unexpected file
  sym1: unexpected symlink
/file1
  content:
    --- expected
    +++ actual
    @@ -1,2 +1 @@
    -not the
     same in both
`, dir.Path())
	assert.Equal(t, result.(cmpFailure).FailureMessage(), expected)
}

type cmpFailure interface {
	FailureMessage() string
}
