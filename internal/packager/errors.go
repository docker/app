package packager

import (
	"fmt"
)

type notFoundError struct {
	message string
}

func (e *notFoundError) Error() string {
	return e.message
}

func newNotFoundError(name string) error {
	return &notFoundError{
		message: fmt.Sprintf("cannot locate application %q on filesystem", name),
	}
}

// IsNotFoundError returns true if the passed object is of the type notFoundError
func IsNotFoundError(err interface{}) bool {
	_, isNotFoundError := err.(*notFoundError)
	return isNotFoundError
}
