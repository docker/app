package bundle

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/deislabs/cnab-go/bundle"
	"github.com/deislabs/cnab-go/bundle/definition"
	"github.com/deislabs/cnab-go/claim"
	"github.com/docker/app/internal"
	"github.com/docker/app/internal/packager"
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
	err := WithFileParameters([]string{tmpDir.Join("params.yaml")})(
		&MergeBundleConfig{
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

	err := WithCommandLineParameters([]string{"param1.param2=value1", "param3=3", "overridden=bar"})(
		&MergeBundleConfig{
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

func prepareBundleWithParameters(b *bundle.Bundle) {
	if b.Parameters != nil {
		return
	}
	b.Parameters = map[string]bundle.Parameter{}
	b.Definitions = definition.Definitions{}
}

func withParameter(name, typ string) bundleOperator {
	return func(b *bundle.Bundle) {
		prepareBundleWithParameters(b)
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
		prepareBundleWithParameters(b)
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
		prepareBundleWithParameters(b)
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
			err := WithOrchestratorParameters("kubernetes", "my-namespace")(&MergeBundleConfig{
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
		first := func(c *MergeBundleConfig) error {
			c.params["param"] = "first"
			return nil
		}
		second := func(c *MergeBundleConfig) error {
			c.params["param"] = "second"
			return nil
		}
		bundle := prepareBundle(withParameterAndDefault("param", "string", "default"))
		i := &store.Installation{Claim: claim.Claim{Bundle: bundle}}
		err := MergeBundleParameters(i,
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
		err := MergeBundleParameters(i)
		assert.NilError(t, err)
		expected := map[string]interface{}{
			"param": "default",
		}
		assert.Assert(t, cmp.DeepEqual(i.Parameters, expected))
	})

	t.Run("Converting values", func(t *testing.T) {
		withIntValue := func(c *MergeBundleConfig) error {
			c.params["param"] = "1"
			return nil
		}
		bundle := prepareBundle(withParameter("param", "integer"))
		i := &store.Installation{Claim: claim.Claim{Bundle: bundle}}
		err := MergeBundleParameters(i, withIntValue)
		assert.NilError(t, err)
		expected := map[string]interface{}{
			"param": 1,
		}
		assert.Assert(t, cmp.DeepEqual(i.Parameters, expected))
	})

	t.Run("Default values", func(t *testing.T) {
		bundle := prepareBundle(withParameterAndDefault("param", "string", "default"))
		i := &store.Installation{Claim: claim.Claim{Bundle: bundle}}
		err := MergeBundleParameters(i)
		assert.NilError(t, err)
		expected := map[string]interface{}{
			"param": "default",
		}
		assert.Assert(t, cmp.DeepEqual(i.Parameters, expected))
	})

	t.Run("Undefined parameter throws warning", func(t *testing.T) {
		withUndefined := func(c *MergeBundleConfig) error {
			c.params["param"] = "1"
			return nil
		}
		bundle := prepareBundle()
		i := &store.Installation{Claim: claim.Claim{Bundle: bundle}}
		buf := new(bytes.Buffer)
		err := MergeBundleParameters(i, withUndefined, WithErrorWriter(buf))
		assert.NilError(t, err)
		assert.Assert(t, strings.Contains(buf.String(), "is not defined in the bundle"))
	})

	t.Run("Warn on undefined parameter", func(t *testing.T) {
		withUndefined := func(c *MergeBundleConfig) error {
			c.params["param"] = "1"
			return nil
		}
		w := bytes.NewBuffer(nil)
		withStdErr := func(c *MergeBundleConfig) error {
			c.stderr = w
			return nil
		}
		bundle := prepareBundle()
		i := &store.Installation{Claim: claim.Claim{Bundle: bundle}}
		err := MergeBundleParameters(i, withUndefined, withStdErr)
		assert.NilError(t, err)
		assert.Equal(t, w.String(), "Warning: parameter \"param\" is not defined in the bundle\n")
	})

	t.Run("Invalid type is rejected", func(t *testing.T) {
		withIntValue := func(c *MergeBundleConfig) error {
			c.params["param"] = "foo"
			return nil
		}
		bundle := prepareBundle(withParameter("param", "integer"))
		i := &store.Installation{Claim: claim.Claim{Bundle: bundle}}
		err := MergeBundleParameters(i, withIntValue)
		assert.ErrorContains(t, err, "invalid value for parameter")
	})

	t.Run("Invalid value is rejected", func(t *testing.T) {
		withInvalidValue := func(c *MergeBundleConfig) error {
			c.params["param"] = "invalid"
			return nil
		}
		bundle := prepareBundle(withParameterAndValues("param", "string", []interface{}{"valid"}))
		i := &store.Installation{Claim: claim.Claim{Bundle: bundle}}
		err := MergeBundleParameters(i, withInvalidValue)
		assert.ErrorContains(t, err, "invalid value for parameter")
	})
}

func TestLabels(t *testing.T) {
	expected := packager.DockerAppArgs{
		Labels: map[string]string{
			"label": "value",
		},
	}
	expectedStr, err := json.Marshal(expected)
	assert.NilError(t, err)

	labels := []string{
		"label=value",
	}
	op := WithLabels(labels)

	config := &MergeBundleConfig{
		bundle: &bundle.Bundle{
			Parameters: map[string]bundle.Parameter{
				internal.ParameterArgs: {},
			},
		},
		params: map[string]string{},
	}
	err = op(config)
	assert.NilError(t, err)
	fmt.Println(config.params)
	l := config.params[internal.ParameterArgs]
	assert.Equal(t, l, string(expectedStr))
}

func TestInvalidLabels(t *testing.T) {
	labels := []string{
		"com.docker.app.label=value",
	}
	op := WithLabels(labels)
	err := op(&MergeBundleConfig{})
	assert.ErrorContains(t, err, fmt.Sprintf("labels cannot start with %q", internal.Namespace))
}
