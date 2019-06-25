package jsonschema

import (
	"encoding/json"
	"strconv"
)

// AllOf MUST be a non-empty array. Each item of the array MUST be a valid JSON Schema.
// An instance validates successfully against this keyword if it validates successfully against all schemas defined by this keyword's value.
type AllOf []*Schema

// NewAllOf creates a new AllOf validator
func NewAllOf() Validator {
	return &AllOf{}
}

// Validate implements the validator interface for AllOf
func (a AllOf) Validate(propPath string, data interface{}, errs *[]ValError) {
	for _, sch := range a {
		sch.Validate(propPath, data, errs)
	}
}

// JSONProp implements JSON property name indexing for AllOf
func (a AllOf) JSONProp(name string) interface{} {
	idx, err := strconv.Atoi(name)
	if err != nil {
		return nil
	}
	if idx > len(a) || idx < 0 {
		return nil
	}
	return a[idx]
}

// JSONChildren implements the JSONContainer interface for AllOf
func (a AllOf) JSONChildren() (res map[string]JSONPather) {
	res = map[string]JSONPather{}
	for i, sch := range a {
		res[strconv.Itoa(i)] = sch
	}
	return
}

// AnyOf MUST be a non-empty array. Each item of the array MUST be a valid JSON Schema.
// An instance validates successfully against this keyword if it validates successfully against at
// least one schema defined by this keyword's value.
type AnyOf []*Schema

// NewAnyOf creates a new AnyOf validator
func NewAnyOf() Validator {
	return &AnyOf{}
}

// Validate implements the validator interface for AnyOf
func (a AnyOf) Validate(propPath string, data interface{}, errs *[]ValError) {
	for _, sch := range a {
		test := &[]ValError{}
		sch.Validate(propPath, data, test)
		if len(*test) == 0 {
			return
		}
	}
	AddError(errs, propPath, data, "did Not match any specified AnyOf schemas")
}

// JSONProp implements JSON property name indexing for AnyOf
func (a AnyOf) JSONProp(name string) interface{} {
	idx, err := strconv.Atoi(name)
	if err != nil {
		return nil
	}
	if idx > len(a) || idx < 0 {
		return nil
	}
	return a[idx]
}

// JSONChildren implements the JSONContainer interface for AnyOf
func (a AnyOf) JSONChildren() (res map[string]JSONPather) {
	res = map[string]JSONPather{}
	for i, sch := range a {
		res[strconv.Itoa(i)] = sch
	}
	return
}

// OneOf MUST be a non-empty array. Each item of the array MUST be a valid JSON Schema.
// An instance validates successfully against this keyword if it validates successfully against exactly one schema defined by this keyword's value.
type OneOf []*Schema

// NewOneOf creates a new OneOf validator
func NewOneOf() Validator {
	return &OneOf{}
}

// Validate implements the validator interface for OneOf
func (o OneOf) Validate(propPath string, data interface{}, errs *[]ValError) {
	matched := false
	for _, sch := range o {
		test := &[]ValError{}
		sch.Validate(propPath, data, test)
		if len(*test) == 0 {
			if matched {
				AddError(errs, propPath, data, "matched more than one specified OneOf schemas")
				return
			}
			matched = true
		}
	}
	if !matched {
		AddError(errs, propPath, data, "did not match any of the specified OneOf schemas")
	}
}

// JSONProp implements JSON property name indexing for OneOf
func (o OneOf) JSONProp(name string) interface{} {
	idx, err := strconv.Atoi(name)
	if err != nil {
		return nil
	}
	if idx > len(o) || idx < 0 {
		return nil
	}
	return o[idx]
}

// JSONChildren implements the JSONContainer interface for OneOf
func (o OneOf) JSONChildren() (res map[string]JSONPather) {
	res = map[string]JSONPather{}
	for i, sch := range o {
		res[strconv.Itoa(i)] = sch
	}
	return
}

// Not MUST be a valid JSON Schema.
// An instance is valid against this keyword if it fails to validate successfully against the schema defined
// by this keyword.
type Not Schema

// NewNot creates a new Not validator
func NewNot() Validator {
	return &Not{}
}

// Validate implements the validator interface for Not
func (n *Not) Validate(propPath string, data interface{}, errs *[]ValError) {
	sch := Schema(*n)
	test := &[]ValError{}
	sch.Validate(propPath, data, test)
	if len(*test) == 0 {
		// TODO - make this error actually make sense
		AddError(errs, propPath, data, "cannot match schema")
	}
}

// JSONProp implements JSON property name indexing for Not
func (n Not) JSONProp(name string) interface{} {
	return Schema(n).JSONProp(name)
}

// JSONChildren implements the JSONContainer interface for Not
func (n Not) JSONChildren() (res map[string]JSONPather) {
	if n.Ref != "" {
		s := Schema(n)
		return map[string]JSONPather{"$ref": &s}
	}
	return Schema(n).JSONChildren()
}

// UnmarshalJSON implements the json.Unmarshaler interface for Not
func (n *Not) UnmarshalJSON(data []byte) error {
	var sch Schema
	if err := json.Unmarshal(data, &sch); err != nil {
		return err
	}
	*n = Not(sch)
	return nil
}

// MarshalJSON implements json.Marshaller for Not
func (n Not) MarshalJSON() ([]byte, error) {
	return json.Marshal(Schema(n))
}
