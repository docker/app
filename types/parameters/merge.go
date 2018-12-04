package parameters

import (
	"github.com/imdario/mergo"
	"github.com/pkg/errors"
)

// Merge merges multiple parameters overriding duplicated keys
func Merge(parameters ...Parameters) (Parameters, error) {
	s := Parameters(map[string]interface{}{})
	for _, setting := range parameters {
		if err := mergo.Merge(&s, setting, mergo.WithOverride, mergo.WithAppendSlice); err != nil {
			return s, errors.Wrap(err, "cannot merge parameters")
		}
	}
	return s, nil
}
