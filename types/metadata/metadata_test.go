package metadata

import (
	"testing"

	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
)

func TestMaintainer(t *testing.T) {
	assert.Check(t, is.Equal(Maintainer{Name: "foo", Email: "foo@bar.com"}.String(), "foo <foo@bar.com>"))
	assert.Check(t, is.Equal(Maintainer{Name: "foo", Email: ""}.String(), "foo"))
	// FIXME(vdemeester) should we validate the mail ?
	assert.Check(t, is.Equal(Maintainer{Name: "foo", Email: "bar"}.String(), "foo <bar>"))
	assert.Check(t, is.Equal(Maintainer{Name: "", Email: ""}.String(), ""))
}

func TestMaintainers(t *testing.T) {
	m1 := Maintainer{Name: "foo", Email: "foo@bar.com"}
	m2 := Maintainer{Name: "bar", Email: "bar@baz.com"}
	assert.Check(t, is.Equal(Maintainers([]Maintainer{}).String(), ""))
	assert.Check(t, is.Equal(Maintainers([]Maintainer{m1}).String(), "foo <foo@bar.com>"))
	assert.Check(t, is.Equal(Maintainers([]Maintainer{m1, m2}).String(), "foo <foo@bar.com>, bar <bar@baz.com>"))
}
