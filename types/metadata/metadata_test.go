package metadata

import (
	"testing"

	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
)

func TestMaintainer(t *testing.T) {
	assert.Check(t, is.Equal(Maintainer{Name: "dev", Email: "dev@example.com"}.String(), "dev <dev@example.com>"))
	assert.Check(t, is.Equal(Maintainer{Name: "dev", Email: ""}.String(), "dev"))
	// FIXME(vdemeester) should we validate the mail ?
	assert.Check(t, is.Equal(Maintainer{Name: "dev", Email: "mail"}.String(), "dev <mail>"))
	assert.Check(t, is.Equal(Maintainer{Name: "", Email: ""}.String(), ""))
}

func TestMaintainers(t *testing.T) {
	m1 := Maintainer{Name: "dev1", Email: "dev1@example.com"}
	m2 := Maintainer{Name: "dev2", Email: "dev2@example.com"}
	assert.Check(t, is.Equal(Maintainers([]Maintainer{}).String(), ""))
	assert.Check(t, is.Equal(Maintainers([]Maintainer{m1}).String(), "dev1 <dev1@example.com>"))
	assert.Check(t, is.Equal(Maintainers([]Maintainer{m1, m2}).String(), "dev1 <dev1@example.com>, dev2 <dev2@example.com>"))
}
