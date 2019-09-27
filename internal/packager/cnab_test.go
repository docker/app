package packager

import (
	"encoding/json"
	"testing"

	"gotest.tools/golden"

	"github.com/docker/app/types"
	"gotest.tools/assert"
)

func TestToCNAB(t *testing.T) {
	app, err := types.NewAppFromDefaultFiles("testdata/packages")
	assert.NilError(t, err)
	actual, err := ToCNAB(app, "test-image")
	assert.NilError(t, err)
	actualJSON, err := json.MarshalIndent(actual, "", "  ")
	assert.NilError(t, err)
	golden.Assert(t, string(actualJSON), "bundle-json.golden")
}
