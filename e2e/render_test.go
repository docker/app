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
		config, resultErr := packager.Render(path.Join("render", app.Name()), overrides, settings, env)
		var result string
		if resultErr == nil {
			var bytes []byte
			bytes, resultErr = yaml.Marshal(config)
			result = string(bytes)
		}
		if resultErr != nil {
			expectedErr := readFile(t, path.Join("render", app.Name(), "expectedError.txt"))
			assert.ErrorContains(t, resultErr, expectedErr)
		} else {
			expectedRender := readFile(t, path.Join("render", app.Name(), "expected.txt"))
			assert.Equal(t, string(expectedRender), result, "rendering missmatch")
		}
	}
}

// readFile returns the content of the file at the designated path normalizing
// line endings by removing any \r.
func readFile(t *testing.T, path string) string {
	content, err := ioutil.ReadFile(path)
	assert.NilError(t, err, "missing '"+path+"' file")
	return strings.Replace(string(content), "\r", "", -1)
}
