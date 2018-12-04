package parameters

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/docker/app/internal/yaml"
	"github.com/pkg/errors"
)

// Parameters represents a parameters map
type Parameters map[string]interface{}

// Flatten returns a flatten view of a parameters
// This becomes a one-level map with keys joined with a dot
func (s Parameters) Flatten() map[string]string {
	return flatten(s)
}

func flatten(s Parameters) map[string]string {
	m := map[string]string{}
	for k, v := range s {
		switch vv := v.(type) {
		case string:
			m[k] = vv
		case map[string]interface{}:
			im := flatten(vv)
			for ik, iv := range im {
				m[k+"."+ik] = iv
			}
		case []string:
			for i, e := range vv {
				m[fmt.Sprintf("%s.%d", k, i)] = fmt.Sprintf("%v", e)
			}
		case []interface{}:
			for i, e := range vv {
				m[fmt.Sprintf("%s.%d", k, i)] = fmt.Sprintf("%v", e)
			}
		default:
			m[k] = fmt.Sprintf("%v", vv)
		}
	}
	return m
}

// FromFlatten takes a flatten map and loads it as a Parameters map
// This uses yaml.Unmarshal to "guess" the type of the value
func FromFlatten(m map[string]string) (Parameters, error) {
	s := map[string]interface{}{}
	for k, v := range m {
		ks := strings.Split(k, ".")
		var converted interface{}
		if err := yaml.Unmarshal([]byte(v), &converted); err != nil {
			return s, err
		}
		if err := assignKey(s, ks, converted); err != nil {
			return s, err
		}
	}
	return Parameters(s), nil
}

func isSupposedSlice(ks []string) (int64, bool) {
	if len(ks) != 1 {
		return 0, false
	}
	i, err := strconv.ParseInt(ks[0], 10, 0)
	return i, err == nil
}

func assignKey(m map[string]interface{}, keys []string, value interface{}) error {
	key := keys[0]
	if len(keys) == 1 {
		if v, present := m[key]; present {
			if reflect.TypeOf(v) != reflect.TypeOf(value) {
				return errors.Errorf("key %s is already present and value has a different type (%T vs %T)", key, v, value)
			}
		}
		m[key] = value
		return nil
	}
	ks := keys[1:]
	if i, ok := isSupposedSlice(ks); ok {
		// it's a slice
		if v, present := m[key]; !present {
			m[key] = make([]interface{}, i+1)
		} else if _, isSlice := v.([]interface{}); !isSlice {
			return errors.Errorf("key %s already present and not a slice (%T)", key, v)
		}
		s := m[key].([]interface{})
		if int64(len(s)) <= i {
			newSlice := make([]interface{}, i+1)
			copy(newSlice, s)
			s = newSlice
		}
		s[i] = value
		m[key] = s
		return nil
	}
	if v, present := m[key]; !present {
		m[key] = map[string]interface{}{}
	} else if _, isMap := v.(map[string]interface{}); !isMap {
		return errors.Errorf("key %s already present and not a map (%T)", key, v)
	}
	return assignKey(m[key].(map[string]interface{}), ks, value)
}
