package render

import (
	"strings"
	"testing"

	"github.com/docker/app/types"
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

func TestRenderEnabledFalse(t *testing.T) {
	for _, tc := range []interface{}{false, "false", "! ${myapp.debug}"} {
		configs := []composetypes.ConfigFile{
			{
				Config: map[string]interface{}{
					"version": "3.7",
					"services": map[string]interface{}{
						"foo": map[string]interface{}{
							"image":     "busybox",
							"command":   []interface{}{"-text", "foo"},
							"x-enabled": tc,
						},
					},
				},
			},
		}
		c, err := render(configs, map[string]string{
			"myapp.debug": "true",
		})
		assert.NilError(t, err)
		assert.Check(t, is.Len(c.Services, 0))
	}
}

func TestValidateBrokenComposeFile(t *testing.T) {
	metadata := strings.NewReader(`version: "0.1"
name: myname`)
	brokenComposeFile := strings.NewReader(`version: "3.6"
unknown-property: value`)
	app := &types.App{Path: "my-app"}
	err := types.Metadata(metadata)(app)
	assert.NilError(t, err)
	err = types.WithComposes(brokenComposeFile)(app)
	assert.NilError(t, err)
	c, err := Render(app, nil)
	assert.Assert(t, is.Nil(c))
	assert.Error(t, err, "failed to load Compose file: unknown-property Additional property unknown-property is not allowed")
}

func TestValidateRenderedApplication(t *testing.T) {
	metadata := strings.NewReader(`version: "0.1"
name: myname`)
	composeFile := strings.NewReader(`
version: "3.6"
services:
    hello:
        image: ${image}`)
	settings := strings.NewReader(`image: hashicorp/http-echo`)
	app := &types.App{Path: "my-app"}
	err := types.Metadata(metadata)(app)
	assert.NilError(t, err)
	err = types.WithComposes(composeFile)(app)
	assert.NilError(t, err)
	err = types.WithSettings(settings)(app)
	assert.NilError(t, err)
	c, err := Render(app, nil)
	assert.Assert(t, c != nil)
	assert.NilError(t, err)
}
