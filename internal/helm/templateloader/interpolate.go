package templateloader

import (
	"strings"

	"github.com/pkg/errors"
)

// should match http://yaml.org/type/bool.html
func toBoolean(value string) (interface{}, error) {
	switch strings.ToLower(value) {
	case "y", "yes", "true", "on":
		return true, nil
	case "n", "no", "false", "off":
		return false, nil
	default:
		return nil, errors.Errorf("invalid boolean: %s", value)
	}
}
