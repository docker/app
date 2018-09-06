package e2e

import (
	"io/ioutil"
	"strings"
	"testing"

	"gotest.tools/assert"
)

func startRegistry(t *testing.T) *Container {
	c := &Container{image: "registry:2", privatePort: 5000}
	c.Start(t)
	return c
}

// readFile returns the content of the file at the designated path normalizing
// line endings by removing any \r.
func readFile(t *testing.T, path string) string {
	t.Helper()
	content, err := ioutil.ReadFile(path)
	assert.NilError(t, err, "missing '"+path+"' file")
	return strings.Replace(string(content), "\r", "", -1)
}

// checkRenderers returns false if appname requires a renderer that is not in enabled
func checkRenderers(appname string, enabled string) bool {
	renderers := []string{"gotemplate", "yatee", "mustache"}
	for _, r := range renderers {
		if strings.Contains(appname, r) && !strings.Contains(enabled, r) {
			return false
		}
	}
	return true
}
