package settings

import (
	"testing"

	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
)

func TestFromFlattenError(t *testing.T) {
	_, err := FromFlatten(map[string]string{
		"foo":     "bar",
		"foo.baz": "biz",
	})
	// Can't have a sub-value (foo.baz) of an existing value (foo)
	assert.Check(t, err != nil)

	_, err = FromFlatten(map[string]string{
		"foo":   "bar",
		"foo.0": "biz",
	})
	// Can't have an array value (foo.0) of an existing value (foo)
	assert.Check(t, err != nil)

	_, err = FromFlatten(map[string]string{
		"foo.0":   "bar",
		"foo.baz": "biz",
	})
	// Can't have an array value (foo.0) and a sub-value (foo.baz) at the same time
	assert.Check(t, err != nil)
}

func TestFromFlatten(t *testing.T) {
	s, err := FromFlatten(map[string]string{
		"foo":       "bar",
		"bar.baz":   "banana",
		"bar.port":  "80",
		"baz.biz.a": "1",
		"baz.biz.b": "2",
		"baz.boz":   "buz",
		"toto.0":    "a",
		"toto.1":    "b",
		"toto.3":    "d",
		"boolean":   "false",
		"frog":      "{bear}",
	})
	assert.NilError(t, err)
	assert.Check(t, is.DeepEqual(s, Settings{
		"foo": "bar",
		"bar": Settings{
			"baz":  "banana",
			"port": 80,
		},
		"baz": Settings{
			"biz": Settings{
				"a": 1,
				"b": 2,
			},
			"boz": "buz",
		},
		"toto":    []interface{}{"a", "b", nil, "d"},
		"boolean": false,
		"frog": map[interface{}]interface{}{
			"bear": nil,
		},
	}))
}
