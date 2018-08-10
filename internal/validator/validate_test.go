package validator

import (
	"testing"

	"gotest.tools/assert"

	"github.com/docker/app/internal"
	"github.com/docker/app/types"
	"gotest.tools/fs"
)

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
	app, err := types.NewAppFromDefaultFiles(dir.Path())
	assert.NilError(t, err)
	err = Validate(app, nil)
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
	app, err := types.NewAppFromDefaultFiles(dir.Path())
	assert.NilError(t, err)
	err = Validate(app, nil)
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
	app, err := types.NewAppFromDefaultFiles(dir.Path())
	assert.NilError(t, err)
	err = Validate(app, nil)
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
	app, err := types.NewAppFromDefaultFiles(dir.Path())
	assert.NilError(t, err)
	err = Validate(app, nil)
	assert.NilError(t, err)
}
