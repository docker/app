package types

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/docker/app/internal"

	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
	"gotest.tools/fs"
)

const (
	validMeta = `name: test-app
version: 0.1.0`
	validCompose = `version: "3.0"
services:
  web:
    image: nginx`
	validSettings = `foo: bar`
)

func TestNewApp(t *testing.T) {
	app, err := NewApp("any-app")
	assert.NilError(t, err)
	assert.Assert(t, is.Equal(app.Path, "any-app"))
}

func TestNewAppFromDefaultFiles(t *testing.T) {
	dir := fs.NewDir(t, "my-app",
		fs.WithFile(internal.MetadataFileName, validMeta),
		fs.WithFile(internal.SettingsFileName, `foo: bar`),
		fs.WithFile(internal.ComposeFileName, validCompose),
	)
	defer dir.Remove()
	app, err := NewAppFromDefaultFiles(dir.Path())
	assert.NilError(t, err)
	assert.Assert(t, is.Len(app.SettingsRaw(), 1))
	assertContentIs(t, app.SettingsRaw()[0], `foo: bar`)
	assert.Assert(t, is.Len(app.Composes(), 1))
	assertContentIs(t, app.Composes()[0], validCompose)
	assertContentIs(t, app.MetadataRaw(), validMeta)
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
		fs.WithFile("my-settings-file", validSettings),
	)
	defer dir.Remove()
	app := &App{Path: "my-app"}
	err := WithSettingsFiles(dir.Join("my-settings-file"))(app)
	assert.NilError(t, err)
	assert.Assert(t, is.Len(app.SettingsRaw(), 1))
	assertContentIs(t, app.SettingsRaw()[0], validSettings)
}

func TestWithSettings(t *testing.T) {
	r := strings.NewReader(validSettings)
	app := &App{Path: "my-app"}
	err := WithSettings(r)(app)
	assert.NilError(t, err)
	assert.Assert(t, is.Len(app.SettingsRaw(), 1))
	assertContentIs(t, app.SettingsRaw()[0], validSettings)
}

func TestWithComposeFilesError(t *testing.T) {
	app := &App{Path: "any-app"}
	err := WithComposeFiles("any-compose-file")(app)
	assert.ErrorContains(t, err, "open any-compose-file")
}

func TestWithComposeFiles(t *testing.T) {
	dir := fs.NewDir(t, "composes",
		fs.WithFile("my-compose-file", validCompose),
	)
	defer dir.Remove()
	app := &App{Path: "my-app"}
	err := WithComposeFiles(dir.Join("my-compose-file"))(app)
	assert.NilError(t, err)
	assert.Assert(t, is.Len(app.Composes(), 1))
	assertContentIs(t, app.Composes()[0], validCompose)
}

func TestWithComposes(t *testing.T) {
	r := strings.NewReader(validCompose)
	app := &App{Path: "my-app"}
	err := WithComposes(r)(app)
	assert.NilError(t, err)
	assert.Assert(t, is.Len(app.Composes(), 1))
	assertContentIs(t, app.Composes()[0], validCompose)
}

func TestMetadataFileError(t *testing.T) {
	app := &App{Path: "any-app"}
	err := MetadataFile("any-metadata-file")(app)
	assert.ErrorContains(t, err, "open any-metadata-file")
}

func TestMetadataFile(t *testing.T) {
	dir := fs.NewDir(t, "metadata",
		fs.WithFile("my-metadata-file", validMeta),
	)
	defer dir.Remove()
	app := &App{Path: "my-app"}
	err := MetadataFile(dir.Join("my-metadata-file"))(app)
	assert.NilError(t, err)
	assert.Assert(t, app.MetadataRaw() != nil)
	assertContentIs(t, app.MetadataRaw(), validMeta)
}

func TestMetadata(t *testing.T) {
	r := strings.NewReader(validMeta)
	app := &App{Path: "my-app"}
	err := Metadata(r)(app)
	assert.NilError(t, err)
	assertContentIs(t, app.MetadataRaw(), validMeta)
}

func assertContentIs(t *testing.T, data []byte, expected string) {
	t.Helper()
	assert.Assert(t, is.Equal(string(data), expected))
}

func TestWithExternalFilesAndNestedDirectories(t *testing.T) {
	dir := fs.NewDir(t, "externalfile",
		fs.WithFile(internal.MetadataFileName, validMeta),
		fs.WithFile(internal.SettingsFileName, `foo: bar`),
		fs.WithFile(internal.ComposeFileName, validCompose),
		fs.WithFile("config.cfg", "something"),
		fs.WithDir("nesteddirectory",
			fs.WithFile("nestedconfig.cfg", "something"),
		),
	)
	defer dir.Remove()
	app, err := NewAppFromDefaultFiles(dir.Path())
	assert.NilError(t, err)
	assert.Assert(t, is.Len(app.ExternalFilePaths(), 2))
	assert.Assert(t, is.Equal(app.ExternalFilePaths()[0], "config.cfg"))
	assert.Assert(t, is.Equal(app.ExternalFilePaths()[1], filepath.Join("nesteddirectory", "nestedconfig.cfg")))
}

func TestExternalFilesAreSorted(t *testing.T) {
	dir := fs.NewDir(t, "externalfile",
		fs.WithFile(internal.MetadataFileName, validMeta),
		fs.WithFile(internal.SettingsFileName, `foo: bar`),
		fs.WithFile(internal.ComposeFileName, validCompose),
		fs.WithFile("c.cfg", "something"),
		fs.WithFile("a.cfg", "something"),
		fs.WithFile("b.cfg", "something"),
		fs.WithDir("nesteddirectory",
			fs.WithFile("a.cfg", "something"),
			fs.WithFile("c.cfg", "something"),
			fs.WithFile("b.cfg", "something"),
		),
	)
	defer dir.Remove()
	app, err := NewAppFromDefaultFiles(dir.Path())
	assert.NilError(t, err)
	assert.Assert(t, is.Len(app.ExternalFilePaths(), 6))
	assert.Assert(t, is.Equal(app.ExternalFilePaths()[0], "a.cfg"))
	assert.Assert(t, is.Equal(app.ExternalFilePaths()[1], "b.cfg"))
	assert.Assert(t, is.Equal(app.ExternalFilePaths()[2], "c.cfg"))
	assert.Assert(t, is.Equal(app.ExternalFilePaths()[3], filepath.Join("nesteddirectory", "a.cfg")))
	assert.Assert(t, is.Equal(app.ExternalFilePaths()[4], filepath.Join("nesteddirectory", "b.cfg")))
	assert.Assert(t, is.Equal(app.ExternalFilePaths()[5], filepath.Join("nesteddirectory", "c.cfg")))
}

func TestWithExternalFilesIncludingNestedCoreFiles(t *testing.T) {
	dir := fs.NewDir(t, "externalfiles",
		fs.WithFile(internal.MetadataFileName, validMeta),
		fs.WithFile(internal.SettingsFileName, `foo: bar`),
		fs.WithFile(internal.ComposeFileName, validCompose),
		fs.WithDir("nesteddirectory",
			fs.WithFile(internal.MetadataFileName, validMeta),
			fs.WithFile(internal.SettingsFileName, `foo: bar`),
			fs.WithFile(internal.ComposeFileName, validCompose),
		),
	)
	defer dir.Remove()
	app, err := NewAppFromDefaultFiles(dir.Path())
	assert.NilError(t, err)
	assert.Assert(t, is.Len(app.ExternalFilePaths(), 3))
	assert.Assert(t, is.Equal(app.ExternalFilePaths()[0], filepath.Join("nesteddirectory", internal.ComposeFileName)))
	assert.Assert(t, is.Equal(app.ExternalFilePaths()[1], filepath.Join("nesteddirectory", internal.MetadataFileName)))
	assert.Assert(t, is.Equal(app.ExternalFilePaths()[2], filepath.Join("nesteddirectory", internal.SettingsFileName)))
}

func TestValidateBrokenMetadata(t *testing.T) {
	r := strings.NewReader(`#version: 0.1.0-missing
name: _INVALID-name
namespace: myHubUsername
maintainers:
    - name: user
      email: user@email.com
    - name: user2
    - name: bad-user
      email: bad-email
unknown: property`)
	app := &App{Path: "my-app"}
	err := Metadata(r)(app)
	assert.Error(t, err, `failed to validate metadata:
- maintainers.2.email: Does not match format 'email'
- name: Does not match format 'hostname'
- version: version is required`)
}

func TestValidateBrokenSettings(t *testing.T) {
	metadata := strings.NewReader(`version: "0.1"
name: myname`)
	composeFile := strings.NewReader(`version: "3.6"`)
	brokenSettings := strings.NewReader(`my-settings:
    1: toto`)
	app := &App{Path: "my-app"}
	err := Metadata(metadata)(app)
	assert.NilError(t, err)
	err = WithComposes(composeFile)(app)
	assert.NilError(t, err)
	err = WithSettings(brokenSettings)(app)
	assert.ErrorContains(t, err, `Non-string key in my-settings: 1`)
}
