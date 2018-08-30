package settings

import (
	"github.com/imdario/mergo"
	"github.com/pkg/errors"
)

// Merge merges multiple settings overriding duplicated keys
func Merge(settings ...Settings) (Settings, error) {
	s := Settings(map[string]interface{}{})
	for _, setting := range settings {
		if err := mergo.Merge(&s, setting, mergo.WithOverride, mergo.WithAppendSlice); err != nil {
			return s, errors.Wrap(err, "cannot merge settings")
		}
	}
	return s, nil
}
