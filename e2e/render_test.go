package e2e

import (
	"io/ioutil"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/docker/lunchbox/packager"

	"github.com/gotestyourself/gotestyourself/assert"
	"gopkg.in/yaml.v2"
)

func TestRender(t *testing.T) {
	apps, err := ioutil.ReadDir("render")
	assert.NilError(t, err, "unable to get apps")
	for _, app := range apps {
		t.Log("testing", app.Name())
		var (
			overrides []string
			settings  []string
		)
		content, err := ioutil.ReadDir(path.Join("render", app.Name()))
		assert.NilError(t, err, "unable to get app: %q", app.Name())
		// look for overrides and settings file to inject in the rendering process
		for _, f := range content {
			split := strings.SplitN(f.Name(), "-", 2)
			if split[0] == "settings" {
				settings = append(settings, path.Join("render", app.Name(), f.Name()))
			}
			if split[0] == "override" {
				overrides = append(overrides, path.Join("render", app.Name(), f.Name()))
			}
		}
		// look for emulated command line env
		env := make(map[string]string)
		if _, err = os.Stat(path.Join("render", app.Name(), "env.yml")); err == nil {
			envRaw, err := ioutil.ReadFile(path.Join("render", app.Name(), "env.yml"))
			assert.NilError(t, err, "unable to read file")
			err = yaml.Unmarshal(envRaw, &env)
			assert.NilError(t, err, "unable to unmarshal env")
		}
		// run the render
		result, resultErr := packager.Render(path.Join("render", app.Name()), overrides, settings, env)
		t.Logf("Render gave %v %v", resultErr, result)
		if resultErr != nil {
			expectedErr, err := ioutil.ReadFile(path.Join("render", app.Name(), "expectedError.txt"))
			assert.NilError(t, err, "unexpected render error: %q", resultErr)
			assert.ErrorContains(t, resultErr, string(expectedErr))
		} else {
			expectedRender, err := ioutil.ReadFile(path.Join("render", app.Name(), "expected.txt"))
			assert.NilError(t, err, "missing 'expected.txt' file")
			assert.Equal(t, string(expectedRender), result, "rendering missmatch")
		}
	}
}
