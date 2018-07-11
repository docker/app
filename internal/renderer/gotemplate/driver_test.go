// +build experimental

package gotemplate

import (
	"testing"

	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
)

const (
	compose = `version: "3.4"
services:
{{if .myapp.enable}}
  enabledservice:
    image: alpine:{{.myapp.alpine_version}}
    command: top
{{end}}
  other:
    image: nginx`
	expectedCompose = `version: "3.4"
services:

  enabledservice:
    image: alpine:3.7
    command: top

  other:
    image: nginx`
)

var (
	settings = map[string]interface{}{
		"myapp": map[string]interface{}{
			"enable":         true,
			"alpine_version": "3.7",
		},
	}
)

func TestDriverErrors(t *testing.T) {
	testCases := []struct {
		name          string
		template      string
		settings      map[string]interface{}
		expectedError string
	}{
		{
			name:          "invalid template",
			template:      "{{}}",
			expectedError: "missing value for command",
		},
		{
			name:          "no-settings",
			template:      compose,
			expectedError: "map has no entry for key",
		},
	}
	d := &Driver{}
	for _, tc := range testCases {
		_, err := d.Apply(tc.template, tc.settings)
		assert.Check(t, err != nil)
		assert.Check(t, is.ErrorContains(err, tc.expectedError))
	}
}

func TestDriver(t *testing.T) {
	d := &Driver{}
	s, err := d.Apply(compose, settings)
	assert.NilError(t, err)
	assert.Check(t, is.Equal(s, expectedCompose))
}
