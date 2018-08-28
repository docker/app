package settings

import (
	"testing"

	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
)

func TestMergeSliceError(t *testing.T) {
	m1 := map[string]interface{}{
		"baz": []string{"a", "b"},
	}
	m2 := map[string]interface{}{
		"baz": []int{1},
	}
	_, err := Merge(m1, m2)
	assert.Check(t, is.ErrorContains(err, "cannot append two slice with different type"))
}

func TestMerge(t *testing.T) {
	m1 := map[string]interface{}{
		"foo": "bar",
		"bar": map[string]interface{}{
			"baz":  "banana",
			"port": "80",
		},
		"baz": []string{"a", "b"},
	}
	m2 := map[string]interface{}{
		"bar": map[string]interface{}{
			"baz":  "biz",
			"port": "10",
			"foo":  "toto",
		},
		"baz":    []string{"c"},
		"banana": "monkey",
	}
	m3, err := FromFlatten(map[string]string{
		"bar.baz": "boz",
	})
	assert.NilError(t, err)
	settings, err := Merge(m1, m2, m3)
	assert.NilError(t, err)
	assert.Check(t, is.DeepEqual(settings.Flatten(), map[string]string{
		"foo":      "bar",
		"bar.baz":  "boz",
		"bar.port": "10",
		"bar.foo":  "toto",
		"baz.0":    "c",
		"banana":   "monkey",
	}))
}
