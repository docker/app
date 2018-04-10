package golden_test

import (
	"testing"

	"github.com/gotestyourself/gotestyourself/assert"
	"github.com/gotestyourself/gotestyourself/golden"
)

var t = &testing.T{}

func ExampleAssert() {
	golden.Assert(t, "foo", "foo-content.golden")
}

func ExampleString() {
	assert.Assert(t, golden.String("foo", "foo-content.golden"))
}

func ExampleAssertBytes() {
	golden.AssertBytes(t, []byte("foo"), "foo-content.golden")
}
