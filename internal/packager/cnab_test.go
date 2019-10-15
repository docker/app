package packager

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/docker/app/internal"
	"github.com/docker/app/types"
	"gotest.tools/assert"
	"gotest.tools/golden"
)

func TestToCNAB(t *testing.T) {
	app, err := types.NewAppFromDefaultFiles("testdata/packages/packing.dockerapp")
	assert.NilError(t, err)
	actual, err := ToCNAB(app, "test-image")
	assert.NilError(t, err)
	actualJSON, err := json.MarshalIndent(actual, "", "  ")
	assert.NilError(t, err)
	s := golden.Get(t, "bundle-json.golden")
	expected := fmt.Sprintf(string(s), internal.Version)
	assert.Equal(t, string(actualJSON), expected)
}
