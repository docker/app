package jsonschema

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"

	"github.com/qri-io/jsonpointer"
)

// Items MUST be either a valid JSON Schema or an array of valid JSON Schemas.
// This keyword determines how child instances validate for arrays, and does not directly validate the
// immediate instance itself.
// * If "Items" is a schema, validation succeeds if all elements in the array successfully validate
//   against that schema.
// * If "Items" is an array of schemas, validation succeeds if each element of the instance validates
//   against the schema at the same position, if any.
// * Omitting this keyword has the same behavior as an empty schema.
type Items struct {
	// need to track weather user specficied a single object or arry
	// b/c it affects AdditionalItems validation semantics
	single  bool
	Schemas []*Schema
}

// NewItems creates a new Items validator
func NewItems() Validator {
	return &Items{}
}

// Validate implements the Validator interface for Items
func (it Items) Validate(propPath string, data interface{}, errs *[]ValError) {
	jp, err := jsonpointer.Parse(propPath)
	if err != nil {
		AddError(errs, propPath, nil, fmt.Sprintf("invalid property path: %s", err.Error()))
	}

	if arr, ok := data.([]interface{}); ok {
		if it.single {
			for i, elem := range arr {
				d, _ := jp.Descendant(strconv.Itoa(i))
				it.Schemas[0].Validate(d.String(), elem, errs)
			}
		} else {
			for i, vs := range it.Schemas {
				if i < len(arr) {
					d, _ := jp.Descendant(strconv.Itoa(i))
					vs.Validate(d.String(), arr[i], errs)
				}
			}
		}
	}
}

// JSONProp implements JSON property name indexing for Items
func (it Items) JSONProp(name string) interface{} {
	idx, err := strconv.Atoi(name)
	if err != nil {
		return nil
	}
	if idx > len(it.Schemas) || idx < 0 {
		return nil
	}
	return it.Schemas[idx]
}

// JSONChildren implements the JSONContainer interface for Items
func (it Items) JSONChildren() (res map[string]JSONPather) {
	res = map[string]JSONPather{}
	for i, sch := range it.Schemas {
		res[strconv.Itoa(i)] = sch
	}
	return
}

// UnmarshalJSON implements the json.Unmarshaler interface for Items
func (it *Items) UnmarshalJSON(data []byte) error {
	s := &Schema{}
	if err := json.Unmarshal(data, s); err == nil {
		*it = Items{single: true, Schemas: []*Schema{s}}
		return nil
	}
	ss := []*Schema{}
	if err := json.Unmarshal(data, &ss); err != nil {
		return err
	}
	*it = Items{Schemas: ss}
	return nil
}

// MarshalJSON implements the json.Marshaler interface for Items
func (it Items) MarshalJSON() ([]byte, error) {
	if it.single {
		return json.Marshal(it.Schemas[0])
	}
	return json.Marshal([]*Schema(it.Schemas))
}

// AdditionalItems determines how child instances validate for arrays, and does not directly validate the immediate
// instance itself.
// If "Items" is an array of schemas, validation succeeds if every instance element at a position greater than
// the size of "Items" validates against "AdditionalItems".
// Otherwise, "AdditionalItems" MUST be ignored, as the "Items" schema (possibly the default value of an empty schema) is applied to all elements.
// Omitting this keyword has the same behavior as an empty schema.
type AdditionalItems struct {
	startIndex int
	Schema     *Schema
}

// NewAdditionalItems creates a new AdditionalItems validator
func NewAdditionalItems() Validator {
	return &AdditionalItems{}
}

// Validate implements the Validator interface for AdditionalItems
func (a *AdditionalItems) Validate(propPath string, data interface{}, errs *[]ValError) {
	jp, err := jsonpointer.Parse(propPath)
	if err != nil {
		AddError(errs, propPath, nil, fmt.Sprintf("invalid property path: %s", err.Error()))
	}

	if a.startIndex >= 0 {
		if arr, ok := data.([]interface{}); ok {
			for i, elem := range arr {
				if i < a.startIndex {
					continue
				}
				d, _ := jp.Descendant(strconv.Itoa(i))
				a.Schema.Validate(d.String(), elem, errs)
			}
		}
	}
}

// JSONProp implements JSON property name indexing for AdditionalItems
func (a *AdditionalItems) JSONProp(name string) interface{} {
	return a.Schema.JSONProp(name)
}

// JSONChildren implements the JSONContainer interface for AdditionalItems
func (a *AdditionalItems) JSONChildren() (res map[string]JSONPather) {
	if a.Schema == nil {
		return map[string]JSONPather{}
	}
	return a.Schema.JSONChildren()
}

// UnmarshalJSON implements the json.Unmarshaler interface for AdditionalItems
func (a *AdditionalItems) UnmarshalJSON(data []byte) error {
	sch := &Schema{}
	if err := json.Unmarshal(data, sch); err != nil {
		return err
	}
	// begin with -1 as default index to prevent AdditionalItems from evaluating
	// unless startIndex is explicitly set
	*a = AdditionalItems{startIndex: -1, Schema: sch}
	return nil
}

// MaxItems MUST be a non-negative integer.
// An array instance is valid against "MaxItems" if its size is less than, or equal to, the value of this keyword.
type MaxItems int

// NewMaxItems creates a new MaxItems validator
func NewMaxItems() Validator {
	return new(MaxItems)
}

// Validate implements the Validator interface for MaxItems
func (m MaxItems) Validate(propPath string, data interface{}, errs *[]ValError) {
	if arr, ok := data.([]interface{}); ok {
		if len(arr) > int(m) {
			AddError(errs, propPath, data, fmt.Sprintf("array length %d exceeds %d max", len(arr), m))
			return
		}
	}
}

// MinItems MUST be a non-negative integer.
// An array instance is valid against "MinItems" if its size is greater than, or equal to, the value of this keyword.
// Omitting this keyword has the same behavior as a value of 0.
type MinItems int

// NewMinItems creates a new MinItems validator
func NewMinItems() Validator {
	return new(MinItems)
}

// Validate implements the Validator interface for MinItems
func (m MinItems) Validate(propPath string, data interface{}, errs *[]ValError) {
	if arr, ok := data.([]interface{}); ok {
		if len(arr) < int(m) {
			AddError(errs, propPath, data, fmt.Sprintf("array length %d below %d minimum items", len(arr), m))
			return
		}
	}
}

// UniqueItems requires array instance elements be unique
// If this keyword has boolean value false, the instance validates successfully. If it has
// boolean value true, the instance validates successfully if all of its elements are unique.
// Omitting this keyword has the same behavior as a value of false.
type UniqueItems bool

// NewUniqueItems creates a new UniqueItems validator
func NewUniqueItems() Validator {
	return new(UniqueItems)
}

// Validate implements the Validator interface for UniqueItems
func (u *UniqueItems) Validate(propPath string, data interface{}, errs *[]ValError) {
	if arr, ok := data.([]interface{}); ok {
		found := []interface{}{}
		for _, elem := range arr {
			for _, f := range found {
				if reflect.DeepEqual(f, elem) {
					AddError(errs, propPath, data, fmt.Sprintf("array items must be unique. duplicated entry: %v", elem))
					return
				}
			}
			found = append(found, elem)
		}
	}
}

// Contains validates that an array instance is valid against "Contains" if at
// least one of its elements is valid against the given schema.
type Contains Schema

// NewContains creates a new Contains validator
func NewContains() Validator {
	return &Contains{}
}

// Validate implements the Validator interface for Contains
func (c *Contains) Validate(propPath string, data interface{}, errs *[]ValError) {
	v := Schema(*c)
	if arr, ok := data.([]interface{}); ok {
		for _, elem := range arr {
			test := &[]ValError{}
			v.Validate(propPath, elem, test)
			if len(*test) == 0 {
				return
			}
		}
		AddError(errs, propPath, data, fmt.Sprintf("must contain at least one of: %v", c))
	}
}

// JSONProp implements JSON property name indexing for Contains
func (c Contains) JSONProp(name string) interface{} {
	return Schema(c).JSONProp(name)
}

// JSONChildren implements the JSONContainer interface for Contains
func (c Contains) JSONChildren() (res map[string]JSONPather) {
	return Schema(c).JSONChildren()
}

// UnmarshalJSON implements the json.Unmarshaler interface for Contains
func (c *Contains) UnmarshalJSON(data []byte) error {
	var sch Schema
	if err := json.Unmarshal(data, &sch); err != nil {
		return err
	}
	*c = Contains(sch)
	return nil
}
