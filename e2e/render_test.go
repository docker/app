package e2e

import (
	"io/ioutil"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/docker/lunchbox/packager"
	"gopkg.in/yaml.v2"
)

func TestRender(t *testing.T) {
	apps, err := ioutil.ReadDir("render")
	if err != nil {
		t.Error(err)
	}
	for _, app := range apps {
		t.Log("testing", app.Name())
		var (
			overrides []string
			settings  []string
		)
		content, err := ioutil.ReadDir(path.Join("render", app.Name()))
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
			if err != nil {
				t.Error(err)
			}
			err = yaml.Unmarshal(envRaw, &env)
		}
		// run the render
		result, resultErr := packager.Render(path.Join("render", app.Name()), overrides, settings, env)
		t.Logf("Render gave %v %v", resultErr, result)
		if resultErr != nil {
			if expectedErr, err := ioutil.ReadFile(path.Join("render", app.Name(), "expectedError.txt")); err == nil {
				if string(expectedErr) != resultErr.Error() {
					t.Errorf("Error message mismatch: expected '%s', got '%s'", expectedErr, resultErr)
				}
			} else {
				t.Errorf("Unexpected render error: '%s'", resultErr)
			}
		} else {
			expectedRender, err := ioutil.ReadFile(path.Join("render", app.Name(), "expected.txt"))
			if err != nil {
				t.Error("Missing 'expected.txt' file")
			} else {
				if string(expectedRender) != result {
					t.Errorf("Rendering mismatch.\n--Expected--\n%s\n--Effective--\n%s", expectedRender, result) 
				}
			}
		}
	}
}