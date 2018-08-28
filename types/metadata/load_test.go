package metadata

import (
	"fmt"
	"testing"

	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
)

func TestInvalidYAML(t *testing.T) {
	_, err := Load([]byte("invalid yaml"))
	assert.Check(t, is.ErrorContains(err, "failed to parse application metadata"))
}

func TestAllFields(t *testing.T) {
	m := AppMetadata{
		Name:        "testapp",
		Version:     "0.2.0",
		Description: "something about this application",
		Namespace:   "testnamespace",
		Maintainers: []Maintainer{
			{
				Name:  "bob",
				Email: "bob@aol.com",
			},
		},
	}
	parsed, err := Load([]byte(fmt.Sprintf(`name: %s
version: %s
description: %s
namespace: %s
maintainers:
  - name: %s
    email: %s
`, m.Name, m.Version, m.Description, m.Namespace, m.Maintainers[0].Name, m.Maintainers[0].Email)))
	assert.NilError(t, err)
	assert.Check(t, is.DeepEqual(parsed, m))
}
