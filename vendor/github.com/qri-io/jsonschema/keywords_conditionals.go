package jsonschema

import (
	"encoding/json"
)

// If MUST be a valid JSON Schema.
// Instances that successfully validate against this keyword's subschema MUST also be valid against the subschema value of the "Then" keyword, if present.
// Instances that fail to validate against this keyword's subschema MUST also be valid against the subschema value of the "Elsee" keyword.
// Validation of the instance against this keyword on its own always succeeds, regardless of the validation outcome of against its subschema.
type If struct {
	Schema Schema
	Then   *Then
	Else   *Else
}

// NewIf allocates a new If validator
func NewIf() Validator {
	return &If{}
}

// Validate implements the Validator interface for If
func (i *If) Validate(propPath string, data interface{}, errs *[]ValError) {
	test := &[]ValError{}
	i.Schema.Validate(propPath, data, test)
	if len(*test) == 0 {
		if i.Then != nil {
			s := Schema(*i.Then)
			sch := &s
			sch.Validate(propPath, data, errs)
			return
		}
	} else {
		if i.Else != nil {
			s := Schema(*i.Else)
			sch := &s
			sch.Validate(propPath, data, errs)
			return
		}
	}
}

// JSONProp implements JSON property name indexing for If
func (i If) JSONProp(name string) interface{} {
	return Schema(i.Schema).JSONProp(name)
}

// JSONChildren implements the JSONContainer interface for If
func (i If) JSONChildren() (res map[string]JSONPather) {
	return i.Schema.JSONChildren()
}

// UnmarshalJSON implements the json.Unmarshaler interface for If
func (i *If) UnmarshalJSON(data []byte) error {
	var sch Schema
	if err := json.Unmarshal(data, &sch); err != nil {
		return err
	}
	*i = If{Schema: sch}
	return nil
}

// MarshalJSON implements json.Marshaler for If
func (i If) MarshalJSON() ([]byte, error) {
	return json.Marshal(i.Schema)
}

// Then MUST be a valid JSON Schema.
// When present alongside of "if", the instance successfully validates against this keyword if it validates against both the "if"'s subschema and this keyword's subschema.
// When "if" is absent, or the instance fails to validate against its subschema, validation against this keyword always succeeds. Implementations SHOULD avoid attempting to validate against the subschema in these cases.
type Then Schema

// NewThen allocates a new Then validator
func NewThen() Validator {
	return &Then{}
}

// Validate implements the Validator interface for Then
func (t *Then) Validate(propPath string, data interface{}, errs *[]ValError) {}

// JSONProp implements JSON property name indexing for Then
func (t Then) JSONProp(name string) interface{} {
	return Schema(t).JSONProp(name)
}

// JSONChildren implements the JSONContainer interface for If
func (t Then) JSONChildren() (res map[string]JSONPather) {
	return Schema(t).JSONChildren()
}

// UnmarshalJSON implements the json.Unmarshaler interface for Then
func (t *Then) UnmarshalJSON(data []byte) error {
	var sch Schema
	if err := json.Unmarshal(data, &sch); err != nil {
		return err
	}
	*t = Then(sch)
	return nil
}

// MarshalJSON implements json.Marshaler for Then
func (t Then) MarshalJSON() ([]byte, error) {
	return json.Marshal(Schema(t))
}

// Else MUST be a valid JSON Schema.
// When present alongside of "if", the instance successfully validates against this keyword if it fails to validate against the "if"'s subschema, and successfully validates against this keyword's subschema.
// When "if" is absent, or the instance successfully validates against its subschema, validation against this keyword always succeeds. Implementations SHOULD avoid attempting to validate against the subschema in these cases.
type Else Schema

// NewElse allocates a new Else validator
func NewElse() Validator {
	return &Else{}
}

// Validate implements the Validator interface for Else
func (e *Else) Validate(propPath string, data interface{}, err *[]ValError) {}

// JSONProp implements JSON property name indexing for Else
func (e Else) JSONProp(name string) interface{} {
	return Schema(e).JSONProp(name)
}

// JSONChildren implements the JSONContainer interface for Else
func (e Else) JSONChildren() (res map[string]JSONPather) {
	return Schema(e).JSONChildren()
}

// UnmarshalJSON implements the json.Unmarshaler interface for Else
func (e *Else) UnmarshalJSON(data []byte) error {
	var sch Schema
	if err := json.Unmarshal(data, &sch); err != nil {
		return err
	}
	*e = Else(sch)
	return nil
}

// MarshalJSON implements json.Marshaler for Else
func (e Else) MarshalJSON() ([]byte, error) {
	return json.Marshal(Schema(e))
}
