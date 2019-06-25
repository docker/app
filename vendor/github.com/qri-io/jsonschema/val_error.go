package jsonschema

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// ValError represents a single error in an instance of a schema
// The only absolutely-required property is Message.
type ValError struct {
	// PropertyPath is a string path that leads to the
	// property that produced the error
	PropertyPath string `json:"propertyPath,omitempty"`
	// InvalidValue is the value that returned the error
	InvalidValue interface{} `json:"invalidValue,omitempty"`
	// RulePath is the path to the rule that errored
	RulePath string `json:"rulePath,omitempty"`
	// Message is a human-readable description of the error
	Message string `json:"message"`
}

// Error implements the error interface for ValError
func (v ValError) Error() string {
	// [propPath]: [value] [message]
	if v.PropertyPath != "" && v.InvalidValue != nil {
		return fmt.Sprintf("%s: %s %s", v.PropertyPath, InvalidValueString(v.InvalidValue), v.Message)
	} else if v.PropertyPath != "" {
		return fmt.Sprintf("%s: %s", v.PropertyPath, v.Message)
	}
	return v.Message
}

// InvalidValueString returns the errored value as a string
func InvalidValueString(data interface{}) string {
	bt, err := json.Marshal(data)
	if err != nil {
		return ""
	}
	bt = bytes.Replace(bt, []byte{'\n', '\r'}, []byte{' '}, -1)
	if MaxValueErrStringLen != -1 && len(bt) > MaxValueErrStringLen {
		bt = append(bt[:MaxValueErrStringLen], []byte("...")...)
	}
	return string(bt)
}

// AddError creates and appends a ValError to errs
func AddError(errs *[]ValError, propPath string, data interface{}, msg string) {
	*errs = append(*errs, ValError{
		PropertyPath: propPath,
		InvalidValue: data,
		Message:      msg,
	})
}
