package definition

import (
	"encoding/base64"
	"fmt"

	"github.com/qri-io/jsonschema"
)

// ContentEncoding represents a "custom" Schema property
type ContentEncoding string

// NewContentEncoding allocates a new ContentEncoding validator
func NewContentEncoding() jsonschema.Validator {
	return new(ContentEncoding)
}

// Validate implements the Validator interface for ContentEncoding
// which, as of writing, isn't included by default in the jsonschema library we consume
func (c ContentEncoding) Validate(propPath string, data interface{}, errs *[]jsonschema.ValError) {
	if obj, ok := data.(string); ok {
		switch c {
		case "base64":
			_, err := base64.StdEncoding.DecodeString(obj)
			if err != nil {
				jsonschema.AddError(errs, propPath, data, fmt.Sprintf("invalid %s value: %s", c, obj))
			}
		// Add validation support for other encodings as needed
		// See https://json-schema.org/latest/json-schema-validation.html#rfc.section.8.3
		default:
			jsonschema.AddError(errs, propPath, data, fmt.Sprintf("unsupported or invalid contentEncoding type of %s", c))
		}
	}
}

// NewRootSchema returns a jsonschema.RootSchema with any needed custom
// jsonschema.Validators pre-registered
func NewRootSchema() *jsonschema.RootSchema {
	// Register custom validators here
	// Note: as of writing, jsonschema doesn't have a stock validator for instances of type `contentEncoding`
	// There may be others missing in the library that exist in http://json-schema.org/draft-07/schema#
	// and thus, we'd need to create/register them here (if not included upstream)
	jsonschema.RegisterValidator("contentEncoding", NewContentEncoding)
	return new(jsonschema.RootSchema)
}
