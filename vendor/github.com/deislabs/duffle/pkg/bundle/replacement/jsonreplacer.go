package replacement

import (
	"encoding/json"
)

// NewJSONReplacer creates a Replacer for JSON documents.
func NewJSONReplacer(indent string) Replacer {
	return jsonReplacer{
		indent: indent,
	}
}

type jsonReplacer struct {
	indent string
}

func (r jsonReplacer) Replace(source string, selector string, value string) (string, error) {
	dict := make(map[string]interface{})
	err := json.Unmarshal([]byte(source), &dict)

	if err != nil {
		return "", err
	}

	selectorPath := parseSelector(selector)
	err = replaceIn(jsonDocMap(dict), selectorPath, value)
	if err != nil {
		return "", err
	}

	bytes, err := json.MarshalIndent(dict, "", r.indent)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

type jsonDocMap map[string]interface{}

func (m jsonDocMap) get(key string) (interface{}, bool) {
	e, ok := m[key]
	return e, ok
}

func (m jsonDocMap) set(key string, value interface{}) {
	m[key] = value
}

func (m jsonDocMap) asInstance(value interface{}) (docmap, bool) {
	if e, ok := value.(map[string]interface{}); ok {
		return jsonDocMap(e), ok
	}
	return jsonDocMap{}, false
}
