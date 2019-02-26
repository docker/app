package commands

import (
	"testing"

	"github.com/deislabs/duffle/pkg/bundle"
	"gotest.tools/assert"
	"gotest.tools/assert/cmp"
	"gotest.tools/fs"
)

func TestWithLoadFiles(t *testing.T) {
	tmpDir := fs.NewDir(t,
		t.Name(),
		fs.WithFile("params.yaml", `param1:
  param2: value1
param3: 3
overridden: bar`))
	defer tmpDir.Remove()

	var bundle *bundle.Bundle
	actual := map[string]string{
		"overridden": "foo",
	}
	err := withFileParameters([]string{tmpDir.Join("params.yaml")})(bundle, actual)
	assert.NilError(t, err)
	expected := map[string]string{
		"param1.param2": "value1",
		"param3":        "3",
		"overridden":    "bar",
	}
	assert.Assert(t, cmp.DeepEqual(actual, expected))
}

func TestWithCommandLineParameters(t *testing.T) {
	var bundle *bundle.Bundle
	actual := map[string]string{
		"overridden": "foo",
	}

	err := withCommandLineParameters([]string{"param1.param2=value1", "param3=3", "overridden=bar"})(bundle, actual)
	assert.NilError(t, err)
	expected := map[string]string{
		"param1.param2": "value1",
		"param3":        "3",
		"overridden":    "bar",
	}
	assert.Assert(t, cmp.DeepEqual(actual, expected))
}

func TestWithOrchestratorParameters(t *testing.T) {
	testCases := []struct {
		name       string
		parameters map[string]bundle.ParameterDefinition
		expected   map[string]string
	}{
		{
			name: "Bundle with orchestrator params",
			parameters: map[string]bundle.ParameterDefinition{
				"docker.orchestrator":         {},
				"docker.kubernetes-namespace": {},
			},
			expected: map[string]string{
				"docker.orchestrator":         "kubernetes",
				"docker.kubernetes-namespace": "my-namespace",
			},
		},
		{
			name:       "Bundle without orchestrator params",
			parameters: map[string]bundle.ParameterDefinition{},
			expected:   map[string]string{},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {

			bundle := &bundle.Bundle{
				Parameters: testCase.parameters,
			}
			actual := map[string]string{}
			err := withOrchestratorParameters("kubernetes", "my-namespace")(bundle, actual)
			assert.NilError(t, err)
			assert.Assert(t, cmp.DeepEqual(actual, testCase.expected))
		})
	}
}

func TestMergeBundleParameters(t *testing.T) {
	t.Run("Override Order", func(t *testing.T) {
		first := func(b *bundle.Bundle, params map[string]string) error {
			params["param"] = "first"
			return nil
		}
		second := func(b *bundle.Bundle, params map[string]string) error {
			params["param"] = "second"
			return nil
		}
		bundle := &bundle.Bundle{
			Parameters: map[string]bundle.ParameterDefinition{
				"param": {
					DefaultValue: "default",
					DataType:     "string",
				},
			},
		}
		actual, err := mergeBundleParameters(bundle,
			first,
			second,
		)

		assert.NilError(t, err)
		expected := map[string]interface{}{
			"param": "second",
		}
		assert.Assert(t, cmp.DeepEqual(actual, expected))
	})

	t.Run("Default values", func(t *testing.T) {
		bundle := &bundle.Bundle{
			Parameters: map[string]bundle.ParameterDefinition{
				"param": {
					DefaultValue: "default",
					DataType:     "string",
				},
			},
		}
		actual, err := mergeBundleParameters(bundle)

		assert.NilError(t, err)
		expected := map[string]interface{}{
			"param": "default",
		}
		assert.Assert(t, cmp.DeepEqual(actual, expected))
	})

	t.Run("Converting values", func(t *testing.T) {
		withIntValue := func(b *bundle.Bundle, params map[string]string) error {
			params["param"] = "1"
			return nil
		}

		bundle := &bundle.Bundle{
			Parameters: map[string]bundle.ParameterDefinition{
				"param": {
					DataType: "int",
				},
			},
		}

		actual, err := mergeBundleParameters(bundle, withIntValue)

		assert.NilError(t, err)
		expected := map[string]interface{}{
			"param": 1,
		}
		assert.Assert(t, cmp.DeepEqual(actual, expected))
	})

	t.Run("Default values", func(t *testing.T) {
		bundle := &bundle.Bundle{
			Parameters: map[string]bundle.ParameterDefinition{
				"param": {
					DefaultValue: "default",
					DataType:     "string",
				},
			},
		}
		actual, err := mergeBundleParameters(bundle)

		assert.NilError(t, err)
		expected := map[string]interface{}{
			"param": "default",
		}
		assert.Assert(t, cmp.DeepEqual(actual, expected))
	})

	t.Run("Undefined parameter is rejected", func(t *testing.T) {
		withUndefined := func(b *bundle.Bundle, params map[string]string) error {
			params["param"] = "1"
			return nil
		}
		bundle := &bundle.Bundle{
			Parameters: map[string]bundle.ParameterDefinition{},
		}
		_, err := mergeBundleParameters(bundle, withUndefined)
		assert.ErrorContains(t, err, "is not defined in the bundle")
	})

	t.Run("Invalid type is rejected", func(t *testing.T) {
		withIntValue := func(b *bundle.Bundle, params map[string]string) error {
			params["param"] = "foo"
			return nil
		}
		bundle := &bundle.Bundle{
			Parameters: map[string]bundle.ParameterDefinition{
				"param": {
					DataType: "int",
				},
			},
		}
		_, err := mergeBundleParameters(bundle, withIntValue)
		assert.ErrorContains(t, err, "invalid value for parameter")
	})

	t.Run("Invalid value is rejected", func(t *testing.T) {
		withIntValue := func(b *bundle.Bundle, params map[string]string) error {
			params["param"] = "invalid"
			return nil
		}
		bundle := &bundle.Bundle{
			Parameters: map[string]bundle.ParameterDefinition{
				"param": {
					DataType:      "string",
					AllowedValues: []interface{}{"valid"},
				},
			},
		}
		_, err := mergeBundleParameters(bundle, withIntValue)
		assert.ErrorContains(t, err, "invalid value for parameter")
	})
}
