package replacement

import "errors"

// Replacer replaces the values of fields matched by a selector.
type Replacer interface {
	Replace(source string, selector string, value string) (string, error)
}

var (
	// ErrSelectorNotFound is reported when the document does not
	// contain a field matching the selector.
	ErrSelectorNotFound = errors.New("Selector not found")
)
