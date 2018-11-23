package parameters

import (
	"github.com/imdario/mergo"
	"github.com/pkg/errors"
)

// Merge merges multiple parameters overriding duplicated keys
func Merge(parameters ...Parameters) (Parameters, error) {
	s := Parameters(map[string]interface{}{})
	for _, parameter := range parameters {
		if err := mergo.Merge(&s, parameter, mergo.WithOverride, mergo.WithAppendSlice); err != nil {
			return s, errors.Wrap(err, "cannot merge parameters")
		}
	}
	return s, nil
}
