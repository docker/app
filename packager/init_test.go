package packager

import (
	"testing"

	"github.com/gotestyourself/gotestyourself/assert"

	"github.com/docker/lunchbox/types"
	yaml "gopkg.in/yaml.v2"
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
