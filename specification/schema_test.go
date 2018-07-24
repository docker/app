package specification

import (
	"testing"

	"gotest.tools/assert"
)

func TestValidateUnknownVersion(t *testing.T) {
	assert.Error(t, Validate(nil, "unknown-version"), "unsupported metadata version: unknown-version")
}

func TestValidateInvalidMetadata(t *testing.T) {
	metadata := map[string]interface{}{
		"name": "_INVALID",
	}
	assert.Error(t, Validate(metadata, "v0.1"),
		`- name: Does not match format 'hostname'
- version: version is required`)
}

func TestValidateMetadata(t *testing.T) {
	metadata := map[string]interface{}{
		"name":    "my-name",
		"version": "my-version",
	}
	assert.NilError(t, Validate(metadata, "v0.1"))
}
