package validator

import (
	"testing"

	"gotest.tools/assert"

	"github.com/docker/app/internal"
	"gotest.tools/fs"
)

func TestValidateMissingFileApplication(t *testing.T) {
	dir := fs.NewDir(t, t.Name(),
		fs.WithDir("no-settings-app", fs.WithFile(internal.MetadataFileName, ""), fs.WithFile(internal.ComposeFileName, "")),
		fs.WithDir("no-metadata-app", fs.WithFile(internal.SettingsFileName, ""), fs.WithFile(internal.ComposeFileName, "")),
		fs.WithDir("no-compose-app", fs.WithFile(internal.MetadataFileName, ""), fs.WithFile(internal.SettingsFileName, "")),
	)
	defer dir.Remove()

	assert.ErrorContains(t, Validate(dir.Join("no-settings-app"), nil, nil), "failed to read application settings")
	assert.ErrorContains(t, Validate(dir.Join("no-metadata-app"), nil, nil), "failed to read application metadata")
	assert.ErrorContains(t, Validate(dir.Join("no-compose-app"), nil, nil), "failed to read application compose")
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
	dir := fs.NewDir(t, t.Name(),
		fs.WithFile(internal.MetadataFileName, brokenMetadata),
		fs.WithFile(internal.ComposeFileName, ""),
		fs.WithFile(internal.SettingsFileName, ""))
	defer dir.Remove()
	err := Validate(dir.Path(), nil, nil)
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
	err := Validate(dir.Path(), nil, nil)
	assert.Error(t, err, `failed to load settings: key 1 in map[interface {}]interface {}{1:"toto"} is not a string`)
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
	err := Validate(dir.Path(), nil, nil)
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
	err := Validate(dir.Path(), nil, nil)
	assert.NilError(t, err)
}
