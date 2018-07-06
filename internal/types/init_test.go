package types

import (
	"testing"

	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
)

func TestNewInitialComposeFile(t *testing.T) {
	f := NewInitialComposeFile()
	assert.Check(t, is.Equal(f.Version, defaultComposefileVersion))
}
