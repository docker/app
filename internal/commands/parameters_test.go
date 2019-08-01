package commands

import (
	"bytes"
	"strings"
	"testing"

	"github.com/deislabs/cnab-go/bundle"
	"github.com/deislabs/cnab-go/bundle/definition"
	"github.com/deislabs/cnab-go/claim"
	"github.com/docker/app/internal"
	"github.com/docker/app/internal/store"
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
	err := withFileParameters([]string{tmpDir.Join("params.yaml")})(
		&mergeBundleConfig{
			bundle: bundle,
			params: actual,
		})
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

	err := withCommandLineParameters([]string{"param1.param2=value1", "param3=3", "overridden=bar"})(
		&mergeBundleConfig{
			bundle: bundle,
			params: actual,
		})
	assert.NilError(t, err)
	expected := map[string]string{
		"param1.param2": "value1",
		"param3":        "3",
		"overridden":    "bar",
	}
	assert.Assert(t, cmp.DeepEqual(actual, expected))
}

type bundleOperator func(*bundle.Bundle)

func withParameter(name, typ string) bundleOperator {
	return func(b *bundle.Bundle) {
		b.Parameters[name] = bundle.Parameter{
			Definition: name,
		}
		b.Definitions[name] = &definition.Schema{
			Type: typ,
		}
	}
}

func withParameterAndDefault(name, typ string, def interface{}) bundleOperator {
	return func(b *bundle.Bundle) {
		b.Parameters[name] = bundle.Parameter{
			Definition: name,
		}
		b.Definitions[name] = &definition.Schema{
			Type:    typ,
			Default: def,
		}
	}
}

func withParameterAndValues(name, typ string, allowedValues []interface{}) bundleOperator {
	return func(b *bundle.Bundle) {
		b.Parameters[name] = bundle.Parameter{
			Definition: name,
		}
		b.Definitions[name] = &definition.Schema{
			Type: typ,
			Enum: allowedValues,
		}
	}
}

func prepareBundle(ops ...bundleOperator) *bundle.Bundle {
	b := &bundle.Bundle{}
	b.Parameters = map[string]bundle.Parameter{}
	b.Definitions = definition.Definitions{}
	for _, op := range ops {
		op(b)
	}
	return b
}

func TestWithOrchestratorParameters(t *testing.T) {
	testCases := []struct {
		name     string
		bundle   *bundle.Bundle
		expected map[string]string
	}{
		{
			name:   "Bundle with orchestrator params",
			bundle: prepareBundle(withParameter(internal.ParameterOrchestratorName, "string"), withParameter(internal.ParameterKubernetesNamespaceName, "string")),
			expected: map[string]string{
				internal.ParameterOrchestratorName:        "kubernetes",
				internal.ParameterKubernetesNamespaceName: "my-namespace",
			},
		},
		{
			name:     "Bundle without orchestrator params",
			bundle:   prepareBundle(),
			expected: map[string]string{},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			actual := map[string]string{}
			err := withOrchestratorParameters("kubernetes", "my-namespace")(&mergeBundleConfig{
				bundle: testCase.bundle,
				params: actual,
			})
			assert.NilError(t, err)
			assert.Assert(t, cmp.DeepEqual(actual, testCase.expected))
		})
	}
}

func TestMergeBundleParameters(t *testing.T) {
	t.Run("Override Order", func(t *testing.T) {
		first := func(c *mergeBundleConfig) error {
			c.params["param"] = "first"
			return nil
		}
		second := func(c *mergeBundleConfig) error {
			c.params["param"] = "second"
			return nil
		}
		bundle := prepareBundle(withParameterAndDefault("param", "string", "default"))
		i := &store.Installation{Claim: claim.Claim{Bundle: bundle}}
		err := mergeBundleParameters(i,
			first,
			second,
		)
		assert.NilError(t, err)
		expected := map[string]interface{}{
			"param": "second",
		}
		assert.Assert(t, cmp.DeepEqual(i.Parameters, expected))
	})

	t.Run("Default values", func(t *testing.T) {
		bundle := prepareBundle(withParameterAndDefault("param", "string", "default"))
		i := &store.Installation{Claim: claim.Claim{Bundle: bundle}}
		err := mergeBundleParameters(i)
		assert.NilError(t, err)
		expected := map[string]interface{}{
			"param": "default",
		}
		assert.Assert(t, cmp.DeepEqual(i.Parameters, expected))
	})

	t.Run("Converting values", func(t *testing.T) {
		withIntValue := func(c *mergeBundleConfig) error {
			c.params["param"] = "1"
			return nil
		}
		bundle := prepareBundle(withParameter("param", "integer"))
		i := &store.Installation{Claim: claim.Claim{Bundle: bundle}}
		err := mergeBundleParameters(i, withIntValue)
		assert.NilError(t, err)
		expected := map[string]interface{}{
			"param": 1,
		}
		assert.Assert(t, cmp.DeepEqual(i.Parameters, expected))
	})

	t.Run("Default values", func(t *testing.T) {
		bundle := prepareBundle(withParameterAndDefault("param", "string", "default"))
		i := &store.Installation{Claim: claim.Claim{Bundle: bundle}}
		err := mergeBundleParameters(i)
		assert.NilError(t, err)
		expected := map[string]interface{}{
			"param": "default",
		}
		assert.Assert(t, cmp.DeepEqual(i.Parameters, expected))
	})

	t.Run("Undefined parameter throws warning", func(t *testing.T) {
		withUndefined := func(c *mergeBundleConfig) error {
			c.params["param"] = "1"
			return nil
		}
		bundle := prepareBundle()
		i := &store.Installation{Claim: claim.Claim{Bundle: bundle}}
		buf := new(bytes.Buffer)
		err := mergeBundleParameters(i, withUndefined, withErrorWriter(buf))
		assert.NilError(t, err)
		assert.Assert(t, strings.Contains(buf.String(), "is not defined in the bundle"))
	})

	t.Run("Undefined parameter with strict mode is rejected", func(t *testing.T) {
		withUndefined := func(c *mergeBundleConfig) error {
			c.params["param"] = "1"
			return nil
		}
		bundle := prepareBundle()
		i := &store.Installation{Claim: claim.Claim{Bundle: bundle}}
		err := mergeBundleParameters(i, withUndefined, withStrictMode(true))
		assert.ErrorContains(t, err, "is not defined in the bundle")
	})

	t.Run("Invalid type is rejected", func(t *testing.T) {
		withIntValue := func(c *mergeBundleConfig) error {
			c.params["param"] = "foo"
			return nil
		}
		bundle := prepareBundle(withParameter("param", "integer"))
		i := &store.Installation{Claim: claim.Claim{Bundle: bundle}}
		err := mergeBundleParameters(i, withIntValue)
		assert.ErrorContains(t, err, "invalid value for parameter")
	})

	t.Run("Invalid value is rejected", func(t *testing.T) {
		withInvalidValue := func(c *mergeBundleConfig) error {
			c.params["param"] = "invalid"
			return nil
		}
		bundle := prepareBundle(withParameterAndValues("param", "string", []interface{}{"valid"}))
		i := &store.Installation{Claim: claim.Claim{Bundle: bundle}}
		err := mergeBundleParameters(i, withInvalidValue)
		assert.ErrorContains(t, err, "invalid value for parameter")
	})
}
