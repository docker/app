package parameters

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/docker/app/internal/yaml"
	"github.com/pkg/errors"
)

// Load loads the given data in parameters
func Load(data []byte, ops ...func(*Options)) (Parameters, error) {
	options := &Options{}
	for _, op := range ops {
		op(options)
	}

	r := bytes.NewReader(data)
	s := make(map[interface{}]interface{})
	decoder := yaml.NewDecoder(r)
	if err := decoder.Decode(&s); err != nil {
		if err == io.EOF {
			return Parameters{}, nil
		}
		return nil, errors.Wrap(err, "failed to read parameters")
	}
	converted, err := convertToStringKeysRecursive(s, "")
	if err != nil {
		return nil, err
	}
	params := converted.(map[string]interface{})
	if options.prefix != "" {
		params = map[string]interface{}{
			options.prefix: params,
		}
	}
	// Make sure params are always loaded expanded
	expandedParams, err := FromFlatten(flatten(params))
	if err != nil {
		return nil, err
	}
	return expandedParams, nil
}

// LoadMultiple loads multiple data in parameters
func LoadMultiple(datas [][]byte, ops ...func(*Options)) (Parameters, error) {
	m := Parameters(map[string]interface{}{})
	for _, data := range datas {
		parameters, err := Load(data, ops...)
		if err != nil {
			return nil, err
		}
		m, err = Merge(m, parameters)
		if err != nil {
			return nil, err
		}
	}
	return m, nil
}

// LoadFile loads a file (path) in parameters (i.e. flatten map)
func LoadFile(path string, ops ...func(*Options)) (Parameters, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Load(data, ops...)
}

// LoadFiles loads multiple path in parameters, merging them.
func LoadFiles(paths []string, ops ...func(*Options)) (Parameters, error) {
	m := Parameters(map[string]interface{}{})
	for _, path := range paths {
		parameters, err := LoadFile(path, ops...)
		if err != nil {
			return nil, err
		}
		m, err = Merge(m, parameters)
		if err != nil {
			return nil, err
		}
	}
	return m, nil
}

// from cli
func convertToStringKeysRecursive(value interface{}, keyPrefix string) (interface{}, error) {
	if mapping, ok := value.(map[interface{}]interface{}); ok {
		dict := make(map[string]interface{})
		for key, entry := range mapping {
			str, ok := key.(string)
			if !ok {
				return nil, formatInvalidKeyError(keyPrefix, key)
			}
			var newKeyPrefix string
			if keyPrefix == "" {
				newKeyPrefix = str
			} else {
				newKeyPrefix = fmt.Sprintf("%s.%s", keyPrefix, str)
			}
			convertedEntry, err := convertToStringKeysRecursive(entry, newKeyPrefix)
			if err != nil {
				return nil, err
			}
			dict[str] = convertedEntry
		}
		return dict, nil
	}
	if list, ok := value.([]interface{}); ok {
		var convertedList []interface{}
		for index, entry := range list {
			newKeyPrefix := fmt.Sprintf("%s[%d]", keyPrefix, index)
			convertedEntry, err := convertToStringKeysRecursive(entry, newKeyPrefix)
			if err != nil {
				return nil, err
			}
			convertedList = append(convertedList, convertedEntry)
		}
		return convertedList, nil
	}
	return value, nil
}

func formatInvalidKeyError(keyPrefix string, key interface{}) error {
	var location string
	if keyPrefix == "" {
		location = "at top level"
	} else {
		location = fmt.Sprintf("in %s", keyPrefix)
	}
	return errors.Errorf("Non-string key %s: %#v", location, key)
}
