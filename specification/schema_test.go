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
	assert.Error(t, Validate(metadata, "v0.2"),
		`- (root): version is required`)
}

func TestValidateMetadata(t *testing.T) {
	metadata := map[string]interface{}{
		"name":    "my-name",
		"version": "my-version",
	}
	assert.NilError(t, Validate(metadata, "v0.2"))
}

func TestValidateMetadataNoName(t *testing.T) {
	metadata := map[string]interface{}{
		//"name":    "my-name",
		// MUST fail! No name
		"version": "my-version",
	}
	assert.Error(t, Validate(metadata, "v0.2"), "- (root): name is required")
}

func TestValidateMetadataNoVersion(t *testing.T) {
	metadata := map[string]interface{}{
		"name": "my-name",
		//"version": "my-version",
		// MUST fail! No version
	}
	assert.Error(t, Validate(metadata, "v0.2"), "- (root): version is required")
}

func TestValidateMetadataV0_2(t *testing.T) {
	metadata := map[string]interface{}{
		"name":    "my-name",
		"version": "my-version",
	}
	assert.NilError(t, Validate(metadata, "v0.2"))
}
