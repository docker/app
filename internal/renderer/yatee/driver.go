// +build experimental

package yatee

import (
	"strings"

	"github.com/docker/app/internal/renderer"
	"github.com/docker/app/internal/yaml"
	"github.com/docker/app/pkg/yatee"
	"github.com/pkg/errors"
)

func init() {
	renderer.Register("yatee", &Driver{})
}

// Driver is the yatee implementation of rendered drivers.
type Driver struct{}

// Apply applies the parameters to the string
func (d *Driver) Apply(s string, parameters map[string]interface{}) (string, error) {
	yateed, err := yatee.Process(s, parameters, yatee.OptionErrOnMissingKey)
	if err != nil {
		return "", err
	}
	m, err := yaml.Marshal(yateed)
	if err != nil {
		return "", errors.Wrap(err, "failed to execute yatee template")
	}
	return strings.Replace(string(m), "$", "$$", -1), nil
}
