package packager

import (
    "github.com/gotestyourself/gotestyourself/assert"
    "testing"

    "gopkg.in/yaml.v2"

    "github.com/docker/lunchbox/types"
)

func TestComposeFileFromScratch(t *testing.T) {
    services := []string{
        "redis", "mysql", "python",
    }

    result, err := composeFileFromScratch(services)
    assert.NilError(t, err)

    expected := types.NewInitialComposeFile()
    expected.Services = &map[string]types.InitialService{
        "redis":  {Image: "redis"},
        "mysql":  {Image: "mysql"},
        "python": {Image: "python"},
    }
    expectedBytes, err := yaml.Marshal(expected)
    assert.NilError(t, err)
    assert.DeepEqual(t, result, expectedBytes)
}
