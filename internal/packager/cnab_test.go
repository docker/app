package packager

import (
	"encoding/json"
	"fmt"
	"regexp"
	"testing"

	"github.com/docker/app/internal"

	"github.com/docker/app/types"
	"gotest.tools/assert"
	"gotest.tools/golden"
)

func TestToCNAB(t *testing.T) {
	app, err := types.NewAppFromDefaultFiles("testdata/packages/packing.dockerapp")
	assert.NilError(t, err)
	actual, err := ToCNAB(app, "test-image")
	assert.NilError(t, err)
	actualJSON, err := json.MarshalIndent(actual, "", "  ")
	assert.NilError(t, err)
	s := golden.Get(t, "bundle-json.golden")
	expectedLiteral := regexp.QuoteMeta(string(s))
	expected := fmt.Sprintf(expectedLiteral, DockerAppPayloadVersionCurrent, internal.Version)
	matches, err := regexp.Match(expected, actualJSON)
	assert.NilError(t, err)
	assert.Assert(t, matches)
}

func TestCNABBundleAPIv1(t *testing.T) {
	const (
		ns     = "com.docker.app."
		cnabNs = "io.cnab."
	)
	//check v1 actions are supported
	requiredActions := []string{
		ns + "inspect",
		ns + "render",
		cnabNs + "status",
	}
	requiredCredentials := []string{
		ns + "registry-creds",
		ns + "context",
	}
	requiredParameters := []string{
		ns + "args",
		ns + "inspect-format",
		ns + "kubernetes-namespace",
		ns + "orchestrator",
		ns + "render-format",
		ns + "share-registry-creds",
	}

	app, err := types.NewAppFromDefaultFiles("testdata/packages/packing.dockerapp")
	assert.NilError(t, err)
	actual, err := ToCNAB(app, "test-image")
	assert.NilError(t, err)
	actualJSON, err := json.MarshalIndent(actual, "", "  ")
	assert.NilError(t, err)

	var expectedJSON map[string]interface{}
	err = json.Unmarshal(actualJSON, &expectedJSON)
	assert.NilError(t, err)

	actions := extract(expectedJSON, "actions")
	for _, action := range requiredActions {
		assert.Equal(t, contains(actions, action), true, fmt.Sprintf("%s not found", action))
	}

	credentials := extract(expectedJSON, "credentials")
	for _, cred := range requiredCredentials {
		assert.Equal(t, contains(credentials, cred), true, fmt.Sprintf("%s not found", cred))
	}

	params := extract(expectedJSON, "parameters")
	for _, param := range requiredParameters {
		assert.Equal(t, contains(params, param), true, fmt.Sprintf("%s not found", param))
	}
}

func extract(data map[string]interface{}, field string) []string {
	var keys []string
	for key := range data[field].(map[string]interface{}) {
		keys = append(keys, key)
	}
	return keys
}

func contains(keyList []string, key string) bool {
	for _, k := range keyList {
		if key == k {
			return true
		}
	}
	return false
}
