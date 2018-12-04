// +build experimental

package yatee

import (
	"testing"

	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
)

const (
	compose = `version: "3.4"
services:
  "@if $myapp.enable":
    enabledservice:
      image: alpine:$myapp.alpine_version
      command: top $$foo
  other:
    image: nginx`
	expectedCompose = `services:
  enabledservice:
    command: top $$foo
    image: alpine:3.7
  other:
    image: nginx
version: "3.4"
`
)

var (
	parameters = map[string]interface{}{
		"myapp": map[string]interface{}{
			"enable":         true,
			"alpine_version": "3.7",
		},
	}
)

func TestDriverErrors(t *testing.T) {
	d := &Driver{}
	_, err := d.Apply("service: $loop", nil)
	assert.Check(t, err != nil)
	assert.Check(t, is.ErrorContains(err, "variable 'loop' not set"))
}

func TestDriver(t *testing.T) {
	d := &Driver{}
	s, err := d.Apply(compose, parameters)
	assert.NilError(t, err)
	t.Log(s)
	assert.Check(t, is.Equal(s, expectedCompose))
}
