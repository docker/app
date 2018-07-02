package renderer

import (
	"testing"

	composetypes "github.com/docker/cli/cli/compose/types"
	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
)

func TestRenderMissingValue(t *testing.T) {
	configFiles := []composetypes.ConfigFile{
		{
			Config: map[string]interface{}{
				"version": "3",
				"services": map[string]interface{}{
					"foo": map[string]interface{}{
						"image": "${imageName}:${version}",
					},
				},
			},
		},
	}
	finalEnv := map[string]string{
		"imageName": "foo",
	}
	_, err := render(configFiles, finalEnv)
	assert.Check(t, err != nil)
	assert.Check(t, is.ErrorContains(err, "required variable"))
}

func TestRender(t *testing.T) {
	configFiles := []composetypes.ConfigFile{
		{
			Config: map[string]interface{}{
				"version": "3",
				"services": map[string]interface{}{
					"foo": map[string]interface{}{
						"image":   "busybox:${version}",
						"command": []interface{}{"-text", "${foo.bar}"},
					},
				},
			},
		},
	}
	finalEnv := map[string]string{
		"version": "latest",
		"foo.bar": "baz",
	}
	c, err := render(configFiles, finalEnv)
	assert.NilError(t, err)
	assert.Check(t, is.Len(c.Services, 1))
	assert.Check(t, is.Equal(c.Services[0].Image, "busybox:latest"))
	assert.Check(t, is.DeepEqual([]string(c.Services[0].Command), []string{"-text", "baz"}))
}
