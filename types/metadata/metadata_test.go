package metadata

import (
	"testing"

	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
)

func TestMaintainer(t *testing.T) {
	assert.Check(t, is.Equal(Maintainer{"foo", "foo@bar.com"}.String(), "foo <foo@bar.com>"))
	assert.Check(t, is.Equal(Maintainer{"foo", ""}.String(), "foo"))
	// FIXME(vdemeester) should we validate the mail ?
	assert.Check(t, is.Equal(Maintainer{"foo", "bar"}.String(), "foo <bar>"))
	assert.Check(t, is.Equal(Maintainer{"", ""}.String(), ""))
}

func TestMaintainers(t *testing.T) {
	m1 := Maintainer{"foo", "foo@bar.com"}
	m2 := Maintainer{"bar", "bar@baz.com"}
	assert.Check(t, is.Equal(Maintainers([]Maintainer{}).String(), ""))
	assert.Check(t, is.Equal(Maintainers([]Maintainer{m1}).String(), "foo <foo@bar.com>"))
	assert.Check(t, is.Equal(Maintainers([]Maintainer{m1, m2}).String(), "foo <foo@bar.com>, bar <bar@baz.com>"))
}
