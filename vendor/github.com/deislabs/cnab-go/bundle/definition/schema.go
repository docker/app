package definition

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

type Definitions map[string]*Schema

// Schema represents a JSON Schema compatible CNAB Definition
type Schema struct {
	Schema               string                 `json:"$schema,omitempty" yaml:"$schema,omitempty"`
	Comment              string                 `json:"$comment,omitempty" yaml:"$comment,omitempty"`
	ID                   string                 `json:"$id,omitempty" yaml:"$id,omitempty"`
	Ref                  string                 `json:"$ref,omitempty" yaml:"$ref,omitempty"`
	AdditionalItems      interface{}            `json:"additionalItems,omitempty" yaml:"additionalItems,omitempty"`
	AdditionalProperties interface{}            `json:"additionalProperties,omitempty" yaml:"additionalProperties,omitempty"`
	AllOf                []*Schema              `json:"allOf,omitempty" yaml:"allOf,omitempty"`
	Const                interface{}            `json:"const,omitempty" yaml:"const,omitempty"`
	Contains             *Schema                `json:"contains,omitempty" yaml:"contains,omitempty"`
	ContentEncoding      string                 `json:"contentEncoding,omitempty" yaml:"contentEncoding,omitempty"`
	ContentMediaType     string                 `json:"contentMediaType,omitempty" yaml:"contentMediaType,omitempty"`
	Default              interface{}            `json:"default,omitempty" yaml:"default,omitempty"`
	Definitions          Definitions            `json:"definitions,omitempty" yaml:"definitions,omitempty"`
	Dependencies         map[string]interface{} `json:"dependencies,omitempty" yaml:"dependencies,omitempty"`
	Description          string                 `json:"description,omitempty" yaml:"description,omitempty"`
	Else                 *Schema                `json:"else,omitempty" yaml:"else,omitempty"`
	Enum                 []interface{}          `json:"enum,omitempty" yaml:"enum,omitempty"`
	Examples             []interface{}          `json:"examples,omitempty" yaml:"examples,omitempty"`
	ExclusiveMaximum     *int                   `json:"exclusiveMaximum,omitempty" yaml:"exclusiveMaximum,omitempty"`
	ExclusiveMinimum     *int                   `json:"exclusiveMinimum,omitempty" yaml:"exclusiveMinimum,omitempty"`
	Format               string                 `json:"format,omitempty" yaml:"format,omitempty"`
	If                   *Schema                `json:"if,omitempty" yaml:"if,omitempty"`
	//Items can be a Schema or an Array of Schema :(
	Items         interface{} `json:"items,omitempty" yaml:"items,omitempty"`
	Maximum       *int        `json:"maximum,omitempty" yaml:"maximum,omitempty"`
	MaxLength     *int        `json:"maxLength,omitempty" yaml:"maxLength,omitempty"`
	MinItems      *int        `json:"minItems,omitempty" yaml:"minItems,omitempty"`
	MinLength     *int        `json:"minLength,omitempty" yaml:"minLength,omitempty"`
	MinProperties *int        `json:"minProperties,omitempty" yaml:"minProperties,omitempty"`
	Minimum       *int        `json:"minimum,omitempty" yaml:"minimum,omitempty"`
	MultipleOf    *int        `json:"multipleOf,omitempty" yaml:"multipleOf,omitempty"`
	Not           *Schema     `json:"not,omitempty" yaml:"not,omitempty"`
	OneOf         *Schema     `json:"oneOf,omitempty" yaml:"oneOf,omitempty"`

	PatternProperties map[string]*Schema `json:"patternProperties,omitempty" yaml:"patternProperties,omitempty"`

	Properties    map[string]*Schema `json:"properties,omitempty" yaml:"properties,omitempty"`
	PropertyNames *Schema            `json:"propertyNames,omitempty" yaml:"propertyNames,omitempty"`
	ReadOnly      *bool              `json:"readOnly,omitempty" yaml:"readOnly,omitempty"`
	Required      []string           `json:"required,omitempty" yaml:"required,omitempty"`
	Then          *Schema            `json:"then,omitempty" yaml:"then,omitempty"`
	Title         string             `json:"title,omitempty" yaml:"title,omitempty"`
	Type          interface{}        `json:"type,omitempty" yaml:"type,omitempty"`
	UniqueItems   *bool              `json:"uniqueItems,omitempty" yaml:"uniqueItems,omitempty"`
	WriteOnly     *bool              `json:"writeOnly,omitempty" yaml:"writeOnly,omitempty"`
}

// GetType will return the singular type for a given schema and a success boolean. If the
// schema does not have a single type, it will return the false boolean and an error.
func (s *Schema) GetType() (string, bool, error) {
	typeString, ok := s.Type.(string)
	if !ok {
		return "", false, errors.Errorf("this schema has multiple types: %v", s.Type)
	}
	return typeString, ok, nil
}

// GetTypes will return the types (as a slice) for a given schema and a success boolean. If the
// schema has a single type, it will return the false boolean and an error.
func (s *Schema) GetTypes() ([]string, bool, error) {
	data, ok := s.Type.([]interface{})
	if !ok {
		return nil, false, errors.Errorf("this schema does not have multiple types: %v", s.Type)
	}
	typeStrings := []string{}
	for _, val := range data {
		typeString, ok := val.(string)
		if !ok {
			return nil, false, errors.Errorf("unknown type value %T", val)
		}
		typeStrings = append(typeStrings, typeString)
	}
	return typeStrings, ok, nil
}

// UnmarshalJSON provides an implementation of a JSON unmarshaler that uses the
// github.com/qri-io/jsonschema to load and validate a given schema. If it is valid,
// then the json is unmarshaled.
func (s *Schema) UnmarshalJSON(data []byte) error {

	// Before we unmarshal into the cnab-go bundle/definition/Schema type, unmarshal into
	// the library struct so we can handle any validation errors in the schema. If there
	// are any errors, return those.
	js := NewRootSchema()
	if err := js.UnmarshalJSON(data); err != nil {
		return err
	}
	// The schema is valid at this point, so now use an indirect wrapper type to actually
	// unmarshal into our type.
	type wrapperType Schema
	wrapper := struct {
		*wrapperType
	}{
		wrapperType: (*wrapperType)(s),
	}
	return json.Unmarshal(data, &wrapper)
}

// ConvertValue attempts to convert the given string value to the type from the
// definition. Note: this is only applicable to string, number, integer and boolean types
func (s *Schema) ConvertValue(val string) (interface{}, error) {
	dataType, ok, err := s.GetType()
	if !ok {
		return nil, errors.Wrapf(err, "unable to determine type: %v", s.Type)
	}
	switch dataType {
	case "string":
		return val, nil
	case "integer":
		return strconv.Atoi(val)
	case "boolean":
		switch strings.ToLower(val) {
		case "true":
			return true, nil
		case "false":
			return false, nil
		default:
			return false, errors.Errorf("%q is not a valid boolean", val)
		}
	default:
		return nil, errors.New("invalid definition")
	}
}
