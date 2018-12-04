package replacement

import (
	yaml "gopkg.in/yaml.v2"
)

// NewYAMLReplacer creates a Replacer for YAML documents.
func NewYAMLReplacer() Replacer {
	return yamlReplacer{}
}

type yamlReplacer struct {
}

func (r yamlReplacer) Replace(source string, selector string, value string) (string, error) {
	dict := make(map[interface{}]interface{})
	err := yaml.Unmarshal([]byte(source), dict)

	if err != nil {
		return "", err
	}

	selectorPath := parseSelector(selector)
	err = replaceIn(yamlDocMap(dict), selectorPath, value)
	if err != nil {
		return "", err
	}

	bytes, err := yaml.Marshal(dict)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

type yamlDocMap map[interface{}]interface{}

func (m yamlDocMap) get(key string) (interface{}, bool) {
	e, ok := m[key]
	return e, ok
}

func (m yamlDocMap) set(key string, value interface{}) {
	m[key] = value
}

func (m yamlDocMap) asInstance(value interface{}) (docmap, bool) {
	if e, ok := value.(map[interface{}]interface{}); ok {
		return yamlDocMap(e), ok
	}
	return yamlDocMap{}, false
}
