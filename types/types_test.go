package types

import (
	"errors"
	"strings"
	"testing"

	"github.com/docker/app/internal"

	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
	"gotest.tools/fs"
)

const (
	yaml = `version: "3.0"
services:
  web:
    image: nginx`
)

func TestNewApp(t *testing.T) {
	app, err := NewApp("any-app")
	assert.NilError(t, err)
	assert.Assert(t, is.Equal(app.Path, "any-app"))
}

func TestNewAppFromDefaultFiles(t *testing.T) {
	dir := fs.NewDir(t, "my-app",
		fs.WithFile(internal.MetadataFileName, "foo"),
		fs.WithFile(internal.SettingsFileName, "foo=bar"),
		fs.WithFile(internal.ComposeFileName, yaml),
	)
	defer dir.Remove()
	app, err := NewAppFromDefaultFiles(dir.Path())
	assert.NilError(t, err)
	assert.Assert(t, is.Len(app.Settings(), 1))
	assertContentIs(t, app.Settings()[0], "foo=bar")
	assert.Assert(t, is.Len(app.Composes(), 1))
	assertContentIs(t, app.Composes()[0], yaml)
	assertContentIs(t, app.Metadata(), "foo")
}

func TestNewAppWithOpError(t *testing.T) {
	_, err := NewApp("any-app", func(_ *App) error { return errors.New("error creating") })
	assert.ErrorContains(t, err, "error creating")
}

func TestWithPath(t *testing.T) {
	app := &App{Path: "any-app"}
	err := WithPath("any-path")(app)
	assert.NilError(t, err)
	assert.Assert(t, is.Equal(app.Path, "any-path"))
}

func TestWithCleanup(t *testing.T) {
	app := &App{Path: "any-app"}
	err := WithCleanup(func() {})(app)
	assert.NilError(t, err)
	assert.Assert(t, app.Cleanup != nil)
}

func TestWithSettingsFilesError(t *testing.T) {
	app := &App{Path: "any-app"}
	err := WithSettingsFiles("any-settings-file")(app)
	assert.ErrorContains(t, err, "open any-settings-file")
}

func TestWithSettingsFiles(t *testing.T) {
	dir := fs.NewDir(t, "settings",
		fs.WithFile("my-settings-file", "foo"),
	)
	defer dir.Remove()
	app := &App{Path: "my-app"}
	err := WithSettingsFiles(dir.Join("my-settings-file"))(app)
	assert.NilError(t, err)
	assert.Assert(t, is.Len(app.Settings(), 1))
	assertContentIs(t, app.Settings()[0], "foo")
}

func TestWithSettings(t *testing.T) {
	r := strings.NewReader("foo")
	app := &App{Path: "my-app"}
	err := WithSettings(r)(app)
	assert.NilError(t, err)
	assert.Assert(t, is.Len(app.Settings(), 1))
	assertContentIs(t, app.Settings()[0], "foo")
}

func TestWithComposeFilesError(t *testing.T) {
	app := &App{Path: "any-app"}
	err := WithComposeFiles("any-compose-file")(app)
	assert.ErrorContains(t, err, "open any-compose-file")
}

func TestWithComposeFiles(t *testing.T) {
	dir := fs.NewDir(t, "composes",
		fs.WithFile("my-compose-file", yaml),
	)
	defer dir.Remove()
	app := &App{Path: "my-app"}
	err := WithComposeFiles(dir.Join("my-compose-file"))(app)
	assert.NilError(t, err)
	assert.Assert(t, is.Len(app.Composes(), 1))
	assertContentIs(t, app.Composes()[0], yaml)
}

func TestWithComposes(t *testing.T) {
	r := strings.NewReader(yaml)
	app := &App{Path: "my-app"}
	err := WithComposes(r)(app)
	assert.NilError(t, err)
	assert.Assert(t, is.Len(app.Composes(), 1))
	assertContentIs(t, app.Composes()[0], yaml)
}

func TestMetadataFileError(t *testing.T) {
	app := &App{Path: "any-app"}
	err := MetadataFile("any-metadata-file")(app)
	assert.ErrorContains(t, err, "open any-metadata-file")
}

func TestMetadataFile(t *testing.T) {
	dir := fs.NewDir(t, "metadata",
		fs.WithFile("my-metadata-file", "foo"),
	)
	defer dir.Remove()
	app := &App{Path: "my-app"}
	err := MetadataFile(dir.Join("my-metadata-file"))(app)
	assert.NilError(t, err)
	assert.Assert(t, app.Metadata() != nil)
	assertContentIs(t, app.Metadata(), "foo")
}

func TestMetadata(t *testing.T) {
	r := strings.NewReader("foo")
	app := &App{Path: "my-app"}
	err := Metadata(r)(app)
	assert.NilError(t, err)
	assertContentIs(t, app.Metadata(), "foo")
}

func assertContentIs(t *testing.T, data []byte, expected string) {
	t.Helper()
	assert.Assert(t, is.Equal(string(data), expected))
}
