package validator

import (
	"testing"

	"gotest.tools/assert"

	"github.com/docker/app/internal"
	"github.com/docker/app/internal/types"
	"gotest.tools/fs"
)

func TestValidateMissingFileApplication(t *testing.T) {
	dir := fs.NewDir(t, t.Name(),
		fs.WithDir("bad-app"),
	)
	defer dir.Remove()
	errs := Validate(types.NewApp(dir.Join("bad-app")), nil)
	assert.ErrorContains(t, errs, "failed to read application settings")
	assert.ErrorContains(t, errs, "failed to read application metadata")
	assert.ErrorContains(t, errs, "failed to read application compose")
}

func TestValidateBrokenMetadata(t *testing.T) {
	brokenMetadata := `#version: 0.1.0-missing
name: _INVALID-name
namespace: myHubUsername
maintainers:
    - name: user
      email: user@email.com
    - name: user2
    - name: bad-user
      email: bad-email
unknown: property`
	composeFile := `version: "3.6"`
	dir := fs.NewDir(t, t.Name(),
		fs.WithFile(internal.MetadataFileName, brokenMetadata),
		fs.WithFile(internal.ComposeFileName, composeFile),
		fs.WithFile(internal.SettingsFileName, ""))
	defer dir.Remove()
	err := Validate(types.NewApp(dir.Path()), nil)
	assert.Error(t, err, `failed to validate metadata:
- maintainers.2.email: Does not match format 'email'
- name: Does not match format 'hostname'
- version: version is required`)
}

func TestValidateBrokenSettings(t *testing.T) {
	metadata := `version: "0.1"
name: myname`
	composeFile := `version: "3.6"`
	brokenSettings := `
my-settings:
    1: toto`
	dir := fs.NewDir(t, t.Name(),
		fs.WithFile(internal.MetadataFileName, metadata),
		fs.WithFile(internal.ComposeFileName, composeFile),
		fs.WithFile(internal.SettingsFileName, brokenSettings))
	defer dir.Remove()
	err := Validate(types.NewApp(dir.Path()), nil)
	assert.ErrorContains(t, err, `Non-string key in my-settings: 1`)
}

func TestValidateBrokenComposeFile(t *testing.T) {
	metadata := `version: "0.1"
name: myname`
	brokenComposeFile := `
version: "3.6"
unknown-property: value`
	dir := fs.NewDir(t, t.Name(),
		fs.WithFile(internal.MetadataFileName, metadata),
		fs.WithFile(internal.ComposeFileName, brokenComposeFile),
		fs.WithFile(internal.SettingsFileName, ""))
	defer dir.Remove()
	err := Validate(types.NewApp(dir.Path()), nil)
	assert.Error(t, err, "failed to load Compose file: unknown-property Additional property unknown-property is not allowed")
}

func TestValidateRenderedApplication(t *testing.T) {
	metadata := `version: "0.1"
name: myname`
	composeFile := `
version: "3.6"
services:
    hello:
        image: ${image}`
	settings := `image: hashicorp/http-echo`
	dir := fs.NewDir(t, t.Name(),
		fs.WithFile(internal.MetadataFileName, metadata),
		fs.WithFile(internal.ComposeFileName, composeFile),
		fs.WithFile(internal.SettingsFileName, settings))
	defer dir.Remove()
	err := Validate(types.NewApp(dir.Path()), nil)
	assert.NilError(t, err)
}
