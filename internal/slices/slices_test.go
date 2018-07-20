package slices

import (
	"testing"

	"gotest.tools/assert"
)

func TestContainsString(t *testing.T) {
	assert.Check(t, !ContainsString(nil, "foo"))
	assert.Check(t, !ContainsString([]string{}, "foo"))
	assert.Check(t, !ContainsString([]string{"foobar"}, "foo"))
	assert.Check(t, ContainsString([]string{"foo", "bar"}, "foo"))
	assert.Check(t, ContainsString([]string{"foo", "bar"}, "bar"))
}
