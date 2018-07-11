// +build experimental

package gotemplate

import (
	"bytes"
	"text/template"

	"github.com/docker/app/internal/renderer"
	"github.com/pkg/errors"
)

func init() {
	renderer.Register("gotemplate", &Driver{})
}

// Driver is the gotemplate implementation of rendered drivers.
type Driver struct{}

// Apply applies the settings to the string
func (d *Driver) Apply(s string, settings map[string]interface{}) (string, error) {
	tmpl, err := template.New("compose").Parse(s)
	if err != nil {
		return "", err
	}
	tmpl.Option("missingkey=error")
	buf := bytes.NewBuffer(nil)
	if err := tmpl.Execute(buf, settings); err != nil {
		return "", errors.Wrap(err, "failed to execute go template")
	}
	return buf.String(), nil
}
