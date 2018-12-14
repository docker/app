package replacement

import (
	"strings"
)

// Abstraction over map to permit generic traversal and substitution
type docmap interface {
	get(key string) (interface{}, bool)
	set(key string, value interface{})
	asInstance(value interface{}) (docmap, bool)
}

func parseSelector(selector string) []string {
	return strings.Split(selector, ".")
}

func replaceIn(dict docmap, selectorPath []string, value string) error {
	entry, ok := dict.get(selectorPath[0])
	if !ok {
		return ErrSelectorNotFound
	}

	if len(selectorPath) == 1 {
		dict.set(selectorPath[0], value)
		return nil
	}

	entryDict, ok := dict.asInstance(entry)
	if !ok {
		return ErrSelectorNotFound // Because we have reached a terminal with some of the selectorPath to go
	}
	rest := selectorPath[1:]

	return replaceIn(entryDict, rest, value)
}
