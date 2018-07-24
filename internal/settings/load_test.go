package settings

import (
	"strings"
	"testing"

	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
	"gotest.tools/fs"
)

func TestLoadErrors(t *testing.T) {
	_, err := Load(strings.NewReader("invalid yaml"))
	assert.Check(t, is.ErrorContains(err, "failed to read settings"))

	_, err = Load(strings.NewReader(`
foo: bar
1: baz`))
	assert.Check(t, is.ErrorContains(err, "Non-string key at top level: 1"))

	_, err = Load(strings.NewReader(`
foo:
  bar: baz
  1: banana`))
	assert.Check(t, is.ErrorContains(err, "Non-string key in foo: 1"))
}

func TestLoad(t *testing.T) {
	settings, err := Load(strings.NewReader(`
foo: bar
bar:
  baz: banana
  port: 80
baz:
  - a
  - b`))
	assert.NilError(t, err)
	assert.Check(t, is.DeepEqual(settings.Flatten(), map[string]string{
		"foo":      "bar",
		"bar.baz":  "banana",
		"bar.port": "80",
		"baz.0":    "a",
		"baz.1":    "b",
	}))
}

func TestLoadWithPrefix(t *testing.T) {
	settings, err := Load(strings.NewReader(`
foo: bar
bar: baz
`), WithPrefix("p"))
	assert.NilError(t, err)
	assert.Check(t, is.DeepEqual(settings.Flatten(), map[string]string{
		"p.foo": "bar",
		"p.bar": "baz",
	}))
}

func TestLoadFiles(t *testing.T) {
	dir := fs.NewDir(t, "files",
		fs.WithFile("s1", `
foo: bar
bar:
  baz: banana
  port: 80`),
		fs.WithFile("s2", `
foo: baz
bar:
  port: 10`),
	)
	defer dir.Remove()

	settings, err := LoadFiles([]string{dir.Join("s1"), dir.Join("s2")})
	assert.NilError(t, err)
	assert.Check(t, is.DeepEqual(settings.Flatten(), map[string]string{
		"foo":      "baz",
		"bar.baz":  "banana",
		"bar.port": "10",
	}))
}
