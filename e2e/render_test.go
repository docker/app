package e2e

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/docker/app/internal"
	"github.com/docker/app/renderer"

	"github.com/gotestyourself/gotestyourself/assert"
	"gopkg.in/yaml.v2"
)

func gather(t *testing.T, dir string) ([]string, []string, map[string]string) {
	var (
		overrides []string
		settings  []string
	)
	content, err := ioutil.ReadDir(dir)
	assert.NilError(t, err, "unable to get app: %q", dir)
	// look for overrides and settings file to inject in the rendering process
	for _, f := range content {
		split := strings.SplitN(f.Name(), "-", 2)
		if split[0] == "settings" {
			settings = append(settings, filepath.Join(dir, f.Name()))
		}
		if split[0] == "override" {
			overrides = append(overrides, filepath.Join(dir, f.Name()))
		}
	}
	// look for emulated command line env
	env := make(map[string]string)
	if _, err = os.Stat(filepath.Join(dir, "env.yml")); err == nil {
		envRaw, err := ioutil.ReadFile(filepath.Join(dir, "env.yml"))
		assert.NilError(t, err, "unable to read file")
		err = yaml.Unmarshal(envRaw, &env)
		assert.NilError(t, err, "unable to unmarshal env")
	}
	return settings, overrides, env
}

// checkRenderer returns false if appname requires a renderer that is not in enabled
func checkRenderers(appname string, enabled string) bool {
	renderers := []string{"gotemplate", "yatee", "mustache"}
	for _, r := range renderers {
		if strings.Contains(appname, r) && !strings.Contains(enabled, r) {
			return false
		}
	}
	return true
}

func checkResult(t *testing.T, result string, resultErr error, dir string) {
	if resultErr != nil {
		ee := filepath.Join(dir, "expectedError.txt")
		if _, err := os.Stat(ee); err != nil {
			assert.NilError(t, resultErr, "unexpected render error")
		}
		expectedErr := readFile(t, ee)
		assert.ErrorContains(t, resultErr, expectedErr)
	} else {
		expectedRender := readFile(t, filepath.Join(dir, "expected.txt"))
		assert.Equal(t, string(expectedRender), result, "rendering missmatch")
	}
}

func TestRender(t *testing.T) {
	apps, err := ioutil.ReadDir("render")
	assert.NilError(t, err, "unable to get apps")
	for _, app := range apps {
		t.Log("testing", app.Name())
		if !checkRenderers(app.Name(), internal.Renderers) {
			t.Log("Required renderer not enabled.")
			continue
		}
		settings, overrides, env := gather(t, filepath.Join("render", app.Name()))
		// run the render
		config, resultErr := renderer.Render(filepath.Join("render", app.Name()), overrides, settings, env)
		var result string
		if resultErr == nil {
			var bytes []byte
			bytes, resultErr = yaml.Marshal(config)
			result = string(bytes)
		}
		checkResult(t, result, resultErr, filepath.Join("render", app.Name()))
	}
}

// readFile returns the content of the file at the designated path normalizing
// line endings by removing any \r.
func readFile(t *testing.T, path string) string {
	content, err := ioutil.ReadFile(path)
	assert.NilError(t, err, "missing '"+path+"' file")
	return strings.Replace(string(content), "\r", "", -1)
}
