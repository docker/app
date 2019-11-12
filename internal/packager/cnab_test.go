package packager

import (
	"encoding/json"
	"fmt"
	"regexp"
	"testing"

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
	expectedLiteral := regexp.QuoteMeta(string(s))
	expected := fmt.Sprintf(expectedLiteral, DockerAppCustomVersionCurrent, `\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}.\d+Z`)
	matches, err := regexp.Match(expected, actualJSON)
	assert.NilError(t, err)
	assert.Assert(t, matches)
}
