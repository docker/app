// Package jsonschema implements draft-handrews-json-schema-validation-00
// JSON Schema (application/schema+json) has several purposes, one of
// which is JSON instance validation. This document specifies a
// vocabulary for JSON Schema to describe the meaning of JSON
// documents, provide hints for user interfaces working with JSON
// data, and to make assertions about what a valid document must look
// like.
package jsonschema

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/qri-io/jsonpointer"
)

// Must turns a JSON string into a *RootSchema, panicing if parsing fails.
// Useful for declaring Schemas in Go code.
func Must(jsonString string) *RootSchema {
	rs := &RootSchema{}
	if err := rs.UnmarshalJSON([]byte(jsonString)); err != nil {
		panic(err)
	}
	return rs
}

// DefaultSchemaPool is a package level map of schemas by identifier
// remote references are cached here.
var DefaultSchemaPool = Definitions{}

// RootSchema is a top-level Schema.
type RootSchema struct {
	Schema
	// The "$schema" keyword is both used as a JSON Schema version
	// identifier and the location of a resource which is itself a JSON
	// Schema, which describes any schema written for this particular
	// version. The value of this keyword MUST be a URI [RFC3986]
	// (containing a scheme) and this URI MUST be normalized. The
	// current schema MUST be valid against the meta-schema identified
	// by this URI. If this URI identifies a retrievable resource, that
	// resource SHOULD be of media type "application/schema+json". The
	// "$schema" keyword SHOULD be used in a root schema. Values for
	// this property are defined in other documents and by other
	// parties. JSON Schema implementations SHOULD implement support
	// for current and previous published drafts of JSON Schema
	// vocabularies as deemed reasonable.
	SchemaURI string `json:"$schema"`
}

// TopLevelType returns a string representing the schema's top-level type.
func (rs *RootSchema) TopLevelType() string {
	if t, ok := rs.Schema.Validators["type"].(*Type); ok {
		return t.String()
	}
	return "unknown"
}

// UnmarshalJSON implements the json.Unmarshaler interface for
// RootSchema
func (rs *RootSchema) UnmarshalJSON(data []byte) error {
	sch := &Schema{}
	if err := json.Unmarshal(data, sch); err != nil {
		return err
	}

	if sch.schemaType == schemaTypeFalse || sch.schemaType == schemaTypeTrue {
		*rs = RootSchema{Schema: *sch}
		return nil
	}

	suri := struct {
		SchemaURI string `json:"$schema"`
	}{}
	if err := json.Unmarshal(data, &suri); err != nil {
		return err
	}

	root := &RootSchema{
		Schema:    *sch,
		SchemaURI: suri.SchemaURI,
	}

	// collect IDs for internal referencing:
	ids := map[string]*Schema{}
	if err := walkJSON(sch, func(elem JSONPather) error {
		if sch, ok := elem.(*Schema); ok {
			if sch.ID != "" {
				ids[sch.ID] = sch
				// For the record, I think this is ridiculous.
				if u, err := url.Parse(sch.ID); err == nil {
					if len(u.Path) >= 1 {
						ids[u.Path[1:]] = sch
					} else if len(u.Fragment) >= 1 {
						// This handles if the identifier is defined as only a fragment (with #)
						// i.e. #/properties/firstName
						// in this case, u.Fragment will have /properties/firstName
						ids[u.Fragment[1:]] = sch
					}
				}
			}
		}
		return nil
	}); err != nil {
		return err
	}

	// pass a pointer to the schema component in here (instead of the
	// RootSchema struct) to ensure root is evaluated for references
	if err := walkJSON(sch, func(elem JSONPather) error {
		if sch, ok := elem.(*Schema); ok {
			if sch.Ref != "" {
				if ids[sch.Ref] != nil {
					sch.ref = ids[sch.Ref]
					return nil
				}

				ptr, err := jsonpointer.Parse(sch.Ref)
				if err != nil {
					return fmt.Errorf("error evaluating json pointer: %s: %s", err.Error(), sch.Ref)
				}
				res, err := root.evalJSONValidatorPointer(ptr)
				if err != nil {
					return err
				}
				if val, ok := res.(Validator); ok {
					sch.ref = val
				} else {
					return fmt.Errorf("%s : %s, %v is not a json pointer to a json schema", sch.Ref, ptr.String(), ptr)
				}
			}
		}
		return nil
	}); err != nil {
		return err
	}

	*rs = RootSchema{
		Schema:    *sch,
		SchemaURI: suri.SchemaURI,
	}
	return nil
}

// FetchRemoteReferences grabs any url-based schema references that
// cannot be locally resolved via network requests
func (rs *RootSchema) FetchRemoteReferences() error {
	sch := &rs.Schema

	refs := DefaultSchemaPool

	if err := walkJSON(sch, func(elem JSONPather) error {
		if sch, ok := elem.(*Schema); ok {
			ref := sch.Ref
			if ref != "" {
				if refs[ref] == nil && ref[0] != '#' {
					if u, err := url.Parse(ref); err == nil {
						if res, err := http.Get(u.String()); err == nil {
							s := &RootSchema{}
							if err := json.NewDecoder(res.Body).Decode(s); err != nil {
								return err
							}
							refs[ref] = &s.Schema
						}
					}
				}

				if refs[ref] != nil {
					sch.ref = refs[ref]
				}
			}
		}
		return nil
	}); err != nil {
		return err
	}

	rs.Schema = *sch
	return nil
}

// ValidateBytes performs schema validation against a slice of json
// byte data
func (rs *RootSchema) ValidateBytes(data []byte) ([]ValError, error) {
	var doc interface{}
	errs := []ValError{}
	if err := json.Unmarshal(data, &doc); err != nil {
		return errs, fmt.Errorf("error parsing JSON bytes: %s", err.Error())
	}
	rs.Validate("/", doc, &errs)
	return errs, nil
}

func (rs *RootSchema) evalJSONValidatorPointer(ptr jsonpointer.Pointer) (res interface{}, err error) {
	res = rs
	for _, token := range ptr {
		if adr, ok := res.(JSONPather); ok {
			res = adr.JSONProp(token)
		} else if !ok {
			err = fmt.Errorf("invalid pointer: %s", ptr)
			return
		}
	}
	return
}

type schemaType int

const (
	schemaTypeObject schemaType = iota
	schemaTypeFalse
	schemaTypeTrue
)

// Schema is the root JSON-schema struct
// A JSON Schema vocabulary is a set of keywords defined for a
// particular purpose. The vocabulary specifies the meaning of its
// keywords as assertions, annotations, and/or any vocabulary-defined
// keyword category.
//
// The two companion standards to this document each define a
// vocabulary: One for instance validation, and one for hypermedia
// annotations.
//
// Vocabularies are the primary mechanism for extensibility within
// the JSON Schema media type. Vocabularies may be defined by any
// entity. Vocabulary authors SHOULD take care to avoid keyword name
// collisions if the vocabulary is intended for broad use, and
// potentially combined with other vocabularies. JSON Schema does not
// provide any formal namespacing system, but also does not constrain
// keyword names, allowing for any number of namespacing approaches.
//
// Vocabularies may build on each other, such as by defining the
// behavior of their keywords with respect to the behavior of
// keywords from another vocabulary,  or by using a keyword from
// another vocabulary with a restricted or expanded set of acceptable
// values. Not all such vocabulary re-use will result in a new
// vocabulary that is compatible with the vocabulary on which it is
// built.
//
// Vocabulary authors SHOULD clearly document what level of
// compatibility, if any, is expected. A schema that itself describes
// a schema is called a meta-schema. Meta-schemas are used to
// validate JSON Schemas and specify which vocabulary it is using.
// [CREF1] A JSON Schema MUST be an object or a boolean.
type Schema struct {
	// internal tracking for true/false/{...} schemas
	schemaType schemaType
	// The "$id" keyword defines a URI for the schema, and the base URI
	// that other URI references within the schema are resolved
	// against. A subschema's "$id" is resolved against the base URI of
	// its parent schema. If no parent sets an explicit base with
	// "$id", the base URI is that of the entire document, as
	// determined per RFC 3986 section 5 [RFC3986].
	ID string `json:"$id,omitempty"`
	// Title and description can be used to decorate a user interface
	// with information about the data produced by this user interface.
	// A title will preferably be short.
	Title string `json:"title,omitempty"`
	// Description provides an explanation about the purpose
	// of the instance described by this schema.
	Description string `json:"description,omitempty"`
	// There are no restrictions placed on the value of this keyword.
	// When multiple occurrences of this keyword are applicable to a
	// single sub-instance, implementations SHOULD remove duplicates.
	// This keyword can be used to supply a default JSON value
	// associated with a particular schema. It is RECOMMENDED that a
	// default value be valid against the associated schema.
	Default interface{} `json:"default,omitempty"`
	// The value of this keyword MUST be an array. There are no
	// restrictions placed on the values within the array. When
	// multiple occurrences of this keyword are applicable to a single
	// sub-instance, implementations MUST provide a flat array of all
	// values rather than an array of arrays. This keyword can be used
	// to provide sample JSON values associated with a particular
	// schema, for the purpose of illustrating usage. It is
	// RECOMMENDED that these values be valid against the associated
	// schema. Implementations MAY use the value(s) of "default", if
	// present, as an additional example. If "examples" is absent,
	// "default" MAY still be used in this manner.
	Examples []interface{} `json:"examples,omitempty"`
	// If "readOnly" has a value of boolean true, it indicates that the
	// value of the instance is managed exclusively by the owning
	// authority, and attempts by an application to modify the value of
	// this property are expected to be ignored or rejected by that
	// owning authority. An instance document that is marked as
	// "readOnly for the entire document MAY be ignored if sent to the
	// owning authority, or MAY result in an error, at the authority's
	// discretion. For example, "readOnly" would be used to mark a
	// database-generated serial number as read-only, while "writeOnly"
	// would be used to mark a password input field. These keywords can
	// be used to assist in user interface instance generation. In
	// particular, an application MAY choose to use a widget that hides
	// input values as they are typed for write-only fields. Omitting
	// these keywords has the same behavior as values of false.
	ReadOnly *bool `json:"readOnly,omitempty"`
	// If "writeOnly" has a value of boolean true, it indicates that
	// the value is never present when the instance is retrieved from
	// the owning authority. It can be present when sent to the owning
	// authority to update or create the document (or the resource it
	// represents), but it will not be included in any updated or newly
	// created version of the instance. An instance document that is
	// marked as "writeOnly" for the entire document MAY be returned as
	// a blank document of some sort, or MAY produce an error upon
	// retrieval, or have the retrieval request ignored, at the
	// authority's discretion.
	WriteOnly *bool `json:"writeOnly,omitempty"`
	// This keyword is reserved for comments from schema authors to
	// readers or maintainers of the schema. The value of this keyword
	// MUST be a string. Implementations MUST NOT present this string
	// to end users. Tools for editing schemas SHOULD support
	// displaying and editing this keyword. The value of this keyword
	// MAY be used in debug or error output which is intended for
	// developers making use of schemas. Schema vocabularies SHOULD
	// allow "$comment" within any object containing vocabulary
	// keywords. Implementations MAY assume "$comment" is allowed
	// unless the vocabulary specifically forbids it. Vocabularies MUST
	// NOT specify any effect of "$comment" beyond what is described in
	// this specification.
	Comment string `json:"$comment,omitempty"`
	// Ref is used to reference a schema, and provides the ability to
	// validate recursive structures through self-reference. An object
	// schema with a "$ref" property MUST be interpreted as a "$ref"
	// reference. The value of the "$ref" property MUST be a URI
	// Reference. Resolved against the current URI base, it identifies
	// the URI of a schema to use. All other properties in a "$ref"
	// object MUST be ignored. The URI is not a network locator, only
	// an identifier. A schema need not be downloadable from the
	// address if it is a network-addressable URL, and implementations
	// SHOULD NOT assume they should perform a network operation when
	// they encounter a network-addressable URI. A schema MUST NOT be
	// run into an infinite loop against a schema. For example, if two
	// schemas "#alice" and "#bob" both have an "allOf" property that
	// refers to the other, a naive validator might get stuck in an
	// infinite recursive loop trying to validate the instance. Schemas
	// SHOULD NOT make use of infinite recursive nesting like this; the
	// behavior is undefined.
	Ref string `json:"$ref,omitempty"`
	// Format functions as both an annotation (Section 3.3) and as an
	// assertion (Section 3.2).
	// While no special effort is required to implement it as an
	// annotation conveying semantic meaning,
	// implementing validation is non-trivial.
	Format string `json:"format,omitempty"`

	ref Validator

	// Definitions provides a standardized location for schema authors
	// to inline re-usable JSON Schemas into a more general schema. The
	// keyword does not directly affect the validation result.
	Definitions Definitions `json:"definitions,omitempty"`

	// TODO - currently a bit of a hack to handle arbitrary JSON data
	// outside the spec
	extraDefinitions Definitions

	Validators map[string]Validator
}

// Path gives a jsonpointer path to the validator
func (s *Schema) Path() string {
	return ""
}

// Validate uses the schema to check an instance, collecting validation
// errors in a slice
func (s *Schema) Validate(propPath string, data interface{}, errs *[]ValError) {
	if s.Ref != "" && s.ref != nil {
		s.ref.Validate(propPath, data, errs)
		return
	} else if s.Ref != "" && s.ref == nil {
		AddError(errs, propPath, data, fmt.Sprintf("%s reference is nil for data: %v", s.Ref, data))
		return
	}

	// TODO - so far all default.json tests pass when no use of
	// "default" is made.
	// Is this correct?

	for _, v := range s.Validators {
		v.Validate(propPath, data, errs)
	}
}

// JSONProp implements the JSONPather for Schema
func (s Schema) JSONProp(name string) interface{} {
	switch name {
	case "$id":
		return s.ID
	case "title":
		return s.Title
	case "description":
		return s.Description
	case "default":
		return s.Default
	case "examples":
		return s.Examples
	case "readOnly":
		return s.ReadOnly
	case "writeOnly":
		return s.WriteOnly
	case "$comment":
		return s.Comment
	case "$ref":
		return s.Ref
	case "definitions":
		return s.Definitions
	case "format":
		return s.Format
	default:
		prop := s.Validators[name]
		if prop == nil && s.extraDefinitions[name] != nil {
			prop = s.extraDefinitions[name]
		}
		return prop
	}
}

// JSONChildren implements the JSONContainer interface for Schema
func (s Schema) JSONChildren() (ch map[string]JSONPather) {
	ch = map[string]JSONPather{}

	if s.extraDefinitions != nil {
		for key, val := range s.extraDefinitions {
			ch[key] = val
		}
	}

	if s.Definitions != nil {
		ch["definitions"] = s.Definitions
	}

	if s.Validators != nil {
		for key, val := range s.Validators {
			if jp, ok := val.(JSONPather); ok {
				ch[key] = jp
			}
		}
	}

	return
}

// _schema is an internal struct for encoding & decoding purposes
type _schema struct {
	ID          string             `json:"$id,omitempty"`
	Title       string             `json:"title,omitempty"`
	Description string             `json:"description,omitempty"`
	Default     interface{}        `json:"default,omitempty"`
	Examples    []interface{}      `json:"examples,omitempty"`
	ReadOnly    *bool              `json:"readOnly,omitempty"`
	WriteOnly   *bool              `json:"writeOnly,omitempty"`
	Comment     string             `json:"$comment,omitempty"`
	Ref         string             `json:"$ref,omitempty"`
	Definitions map[string]*Schema `json:"definitions,omitempty"`
	Format      string             `json:"format,omitempty"`
}

// UnmarshalJSON implements the json.Unmarshaler interface for Schema
func (s *Schema) UnmarshalJSON(data []byte) error {
	// support simple true false schemas that always pass or fail
	var b bool
	if err := json.Unmarshal(data, &b); err == nil {
		if b {
			// boolean true Always passes validation, as if the empty schema {}
			*s = Schema{schemaType: schemaTypeTrue}
			return nil
		}
		// boolean false Always fails validation, as if the schema { "not":{} }
		*s = Schema{schemaType: schemaTypeFalse, Validators: map[string]Validator{"not": &Not{}}}
		return nil
	}

	_s := _schema{}
	if err := json.Unmarshal(data, &_s); err != nil {
		return err
	}

	sch := &Schema{
		ID:          _s.ID,
		Title:       _s.Title,
		Description: _s.Description,
		Default:     _s.Default,
		Examples:    _s.Examples,
		ReadOnly:    _s.ReadOnly,
		WriteOnly:   _s.WriteOnly,
		Comment:     _s.Comment,
		Ref:         _s.Ref,
		Definitions: _s.Definitions,
		Format:      _s.Format,
		Validators:  map[string]Validator{},
	}

	// if a reference is present everything else is *supposed to be* ignored
	// but the tests seem to require that this is not the case
	// I'd like to do this:
	// if sch.Ref != "" {
	// 	*s = Schema{Ref: sch.Ref}
	// 	return nil
	// }
	// but returning the full struct makes tests pass, because things like
	// testdata/draft7/ref.json#/4/schema
	// mean we should return the full object

	valprops := map[string]json.RawMessage{}
	if err := json.Unmarshal(data, &valprops); err != nil {
		return err
	}

	for prop, rawmsg := range valprops {
		var val Validator
		if mk, ok := DefaultValidators[prop]; ok {
			val = mk()
		} else {
			switch prop {
			// skip any already-parsed props
			case "$schema", "$id", "title", "description", "default", "examples", "readOnly", "writeOnly", "$comment", "$ref", "definitions", "format":
				continue
			default:
				// assume non-specified props are "extra definitions"
				if sch.extraDefinitions == nil {
					sch.extraDefinitions = Definitions{}
				}
				s := new(Schema)
				if err := json.Unmarshal(rawmsg, s); err != nil {
					return fmt.Errorf("error unmarshaling %s from json: %s", prop, err.Error())
				}
				sch.extraDefinitions[prop] = s
				continue
			}
		}
		if err := json.Unmarshal(rawmsg, val); err != nil {
			return fmt.Errorf("error unmarshaling %s from json: %s", prop, err.Error())
		}
		sch.Validators[prop] = val
	}

	if sch.Validators["if"] != nil {
		if ite, ok := sch.Validators["if"].(*If); ok {
			if s, ok := sch.Validators["then"].(*Then); ok {
				ite.Then = s
			}
			if s, ok := sch.Validators["else"].(*Else); ok {
				ite.Else = s
			}
		}
	}

	// TODO - replace all these assertions with methods on Schema that return proper types
	if sch.Validators["items"] != nil && sch.Validators["additionalItems"] != nil && !sch.Validators["items"].(*Items).single {
		sch.Validators["additionalItems"].(*AdditionalItems).startIndex = len(sch.Validators["items"].(*Items).Schemas)
	}
	if sch.Validators["properties"] != nil && sch.Validators["additionalProperties"] != nil {
		sch.Validators["additionalProperties"].(*AdditionalProperties).Properties = sch.Validators["properties"].(*Properties)
	}
	if sch.Validators["patternProperties"] != nil && sch.Validators["additionalProperties"] != nil {
		sch.Validators["additionalProperties"].(*AdditionalProperties).patterns = sch.Validators["patternProperties"].(*PatternProperties)
	}

	*s = Schema(*sch)
	return nil
}

// MarshalJSON implements the json.Marshaler interface for Schema
func (s Schema) MarshalJSON() ([]byte, error) {
	switch s.schemaType {
	case schemaTypeFalse:
		return []byte("false"), nil
	case schemaTypeTrue:
		return []byte("true"), nil
	default:
		obj := map[string]interface{}{}

		if s.ID != "" {
			obj["$id"] = s.ID
		}
		if s.Title != "" {
			obj["title"] = s.Title
		}
		if s.Description != "" {
			obj["description"] = s.Description
		}
		if s.Default != nil {
			obj["default"] = s.Default
		}
		if s.Examples != nil {
			obj["examples"] = s.Examples
		}
		if s.ReadOnly != nil {
			obj["readOnly"] = s.ReadOnly
		}
		if s.WriteOnly != nil {
			obj["writeOnly"] = s.WriteOnly
		}
		if s.Comment != "" {
			obj["$comment"] = s.Comment
		}
		if s.Ref != "" {
			obj["$ref"] = s.Ref
		}
		if s.Definitions != nil {
			obj["definitions"] = s.Definitions
		}
		if s.Format != "" {
			obj["format"] = s.Format
		}
		if s.Definitions != nil {
			obj["definitions"] = s.Definitions
		}

		for k, v := range s.Validators {
			obj[k] = v
		}
		for k, v := range s.extraDefinitions {
			obj[k] = v
		}
		return json.Marshal(obj)
	}
}

// Definitions implements a map of schemas while also satsfying the JSON
// traversal methods
type Definitions map[string]*Schema

// JSONProp implements the JSONPather for Definitions
func (d Definitions) JSONProp(name string) interface{} {
	return d[name]
}

// JSONChildren implements the JSONContainer interface for Definitions
func (d Definitions) JSONChildren() (r map[string]JSONPather) {
	r = map[string]JSONPather{}
	for key, val := range d {
		r[key] = val
	}
	return
}
