package packager

import (
	"os"
	"testing"

	"github.com/docker/app/types"
	"gotest.tools/assert"
)

func TestCnabParameterMapping(t *testing.T) {
	f, err := os.Open("testdata/parameters.yml")
	assert.NilError(t, err)
	defer f.Close()
	app := &types.App{
		Name: "test",
	}
	assert.NilError(t, types.WithParameters(f)(app))
	result := ExtractCNABParameterMapping(app.Parameters())

	expectedParameterToCnabEnv := map[string]string{
		"aa":       "docker_param1",
		"bb.bb":    "docker_param2",
		"bb.cc":    "docker_param3",
		"cc.aa.dd": "docker_param4",
		"cc.bb":    "docker_param5",
		"cc.cc":    "docker_param6",
	}
	expectedEnvToParameter := map[string]string{}
	for k, v := range expectedParameterToCnabEnv {
		expectedEnvToParameter[v] = k
	}

	assert.DeepEqual(t, result.ParameterToCNABEnv, expectedParameterToCnabEnv)
	assert.DeepEqual(t, result.CNABEnvToParameter, expectedEnvToParameter)
}

func TestCnabParameterExtraction(t *testing.T) {
	env := []string{
		"docker_param1=val",
		"aa=should_not_bind",
		"docker_param2=override1",
		"docker_param3=override2",
		"docker_param4=override3",
		"docker_param5=override4",
		"docker_param6=override5",
	}
	f, err := os.Open("testdata/parameters.yml")
	assert.NilError(t, err)
	defer f.Close()
	app := &types.App{
		Name: "test",
	}
	assert.NilError(t, types.WithParameters(f)(app))
	mapping := ExtractCNABParameterMapping(app.Parameters())
	result := ExtractCNABParametersValues(mapping, env)
	expectedValues := map[string]string{
		"aa":       "val",
		"bb.bb":    "override1",
		"bb.cc":    "override2",
		"cc.aa.dd": "override3",
		"cc.bb":    "override4",
		"cc.cc":    "override5",
	}
	assert.DeepEqual(t, result, expectedValues)
}
