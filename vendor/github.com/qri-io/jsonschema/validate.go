package jsonschema

// MaxValueErrStringLen sets how long a value can be before it's length is truncated
// when printing error strings
// a special value of -1 disables output trimming
var MaxValueErrStringLen = 20

// Validator is an interface for anything that can validate.
// JSON-Schema keywords are all examples of validators
type Validator interface {
	// Validate checks decoded JSON data and writes
	// validation errors (if any) to an outparam slice of ValErrors
	// propPath indicates the position of data in the json tree
	Validate(propPath string, data interface{}, errs *[]ValError)
}

// BaseValidator is a foundation for building a validator
type BaseValidator struct {
	path string
}

// SetPath sets base validator's path
func (b *BaseValidator) SetPath(path string) {
	b.path = path
}

// Path gives this validator's path
func (b BaseValidator) Path() string {
	return b.path
}

// AddError is a convenience method for appending a new error to an existing error slice
func (b BaseValidator) AddError(errs *[]ValError, propPath string, data interface{}, msg string) {
	*errs = append(*errs, ValError{
		PropertyPath: propPath,
		RulePath:     b.Path(),
		InvalidValue: data,
		Message:      msg,
	})
}

// ValMaker is a function that generates instances of a validator.
// Calls to ValMaker will be passed directly to json.Marshal,
// so the returned value should be a pointer
type ValMaker func() Validator

// RegisterValidator adds a validator to DefaultValidators.
// Custom Validators should satisfy the validator interface,
// and be able to get cleanly endcode/decode to JSON
func RegisterValidator(propName string, maker ValMaker) {
	// TODO - should this call the function and panic if
	// the result can't be fed to json.Umarshal?
	DefaultValidators[propName] = maker
}

// DefaultValidators is a map of JSON keywords to Validators
// to draw from when decoding schemas
var DefaultValidators = map[string]ValMaker{
	// standard keywords
	"type":  NewType,
	"enum":  NewEnum,
	"const": NewConst,

	// numeric keywords
	"multipleOf":       NewMultipleOf,
	"maximum":          NewMaximum,
	"exclusiveMaximum": NewExclusiveMaximum,
	"minimum":          NewMinimum,
	"exclusiveMinimum": NewExclusiveMinimum,

	// string keywords
	"maxLength": NewMaxLength,
	"minLength": NewMinLength,
	"pattern":   NewPattern,

	// boolean keywords
	"allOf": NewAllOf,
	"anyOf": NewAnyOf,
	"oneOf": NewOneOf,
	"not":   NewNot,

	// array keywords
	"items":           NewItems,
	"additionalItems": NewAdditionalItems,
	"maxItems":        NewMaxItems,
	"minItems":        NewMinItems,
	"uniqueItems":     NewUniqueItems,
	"contains":        NewContains,

	// object keywords
	"maxProperties":        NewMaxProperties,
	"minProperties":        NewMinProperties,
	"required":             NewRequired,
	"properties":           NewProperties,
	"patternProperties":    NewPatternProperties,
	"additionalProperties": NewAdditionalProperties,
	"dependencies":         NewDependencies,
	"propertyNames":        NewPropertyNames,

	// conditional keywords
	"if":   NewIf,
	"then": NewThen,
	"else": NewElse,

	//optional formats
	"format": NewFormat,
}
