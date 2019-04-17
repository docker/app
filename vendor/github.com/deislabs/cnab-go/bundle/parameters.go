package bundle

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// ParameterDefinition defines a single parameter for a CNAB bundle
type ParameterDefinition struct {
	DataType      string             `json:"type" mapstructure:"type"`
	DefaultValue  interface{}        `json:"defaultValue,omitempty" mapstructure:"defaultValue"`
	AllowedValues []interface{}      `json:"allowedValues,omitempty" mapstructure:"allowedValues"`
	Required      bool               `json:"required,omitempty" mapstructure:"required"`
	MinValue      *int               `json:"minValue,omitempty" mapstructure:"minValue"`
	MaxValue      *int               `json:"maxValue,omitempty" mapstructure:"maxValue"`
	MinLength     *int               `json:"minLength,omitempty" mapstructure:"minLength"`
	MaxLength     *int               `json:"maxLength,omitempty" mapstructure:"maxLength"`
	Metadata      *ParameterMetadata `json:"metadata,omitempty" mapstructure:"metadata"`
	Destination   *Location          `json:"destination,omitemtpty" mapstructure:"destination"`
	ApplyTo       []string           `json:"apply-to,omitempty" mapstructure:"apply-to,omitempty"`
}

// ParameterMetadata contains metadata for a parameter definition.
type ParameterMetadata struct {
	Description string `json:"description,omitempty" mapstructure:"description"`
}

// ValidateParameterValue checks whether a value is valid as the value of
// the specified parameter.
func (pd ParameterDefinition) ValidateParameterValue(value interface{}) error {
	if err := pd.validateByType(value); err != nil {
		return err
	}

	return pd.validateAllowedValue(value)
}
func (pd ParameterDefinition) validateByType(value interface{}) error {
	switch pd.DataType {
	case "string":
		return pd.validateStringParameterValue(value)
	case "int":
		return pd.validateIntParameterValue(value)
	case "bool":
		return pd.validateBoolParameterValue(value)
	default:
		return errors.New("invalid parameter definition")
	}
}

func (pd ParameterDefinition) validateAllowedValue(value interface{}) error {
	if len(pd.AllowedValues) > 0 {
		val := pd.CoerceValue(value)
		if !isInCollection(val, pd.allowedValues()) {
			return errors.New("value is not in the set of allowed values for this parameter")
		}
	}
	return nil
}

func (pd ParameterDefinition) allowedValues() []interface{} {
	if pd.DataType == "int" {
		return intify(pd.AllowedValues)
	}
	return pd.AllowedValues
}

// "Allowed value" numeric collections loaded from JSON will be materialised
// by Go as float64.  We support only ints and so want to treat them as such.
func intify(values []interface{}) []interface{} {
	result := []interface{}{}
	for _, v := range values {
		f, ok := v.(float64)
		if ok {
			result = append(result, int(f))
		} else {
			result = append(result, v)
		}
	}
	return result
}

// CoerceValue coerces the given value to the definition's DataType;
// unlike ConvertValue, which performs string parsing, it assumes the
// value is already of a suitable type (and validated)
func (pd ParameterDefinition) CoerceValue(value interface{}) interface{} {
	if pd.DataType == "int" {
		f, ok := value.(float64)
		if ok {
			i, ok := asInt(f)
			if !ok {
				return f
			}
			return i
		}
	}
	return value
}

// ConvertValue tries to convert the given value to the definition's DataType
//
// It will return an error if it cannot be converted
func (pd ParameterDefinition) ConvertValue(val string) (interface{}, error) {
	switch pd.DataType {
	case "string":
		return val, nil
	case "int":
		return strconv.Atoi(val)
	case "bool":
		if strings.ToLower(val) == "true" {
			return true, nil
		} else if strings.ToLower(val) == "false" {
			return false, nil
		} else {
			return false, fmt.Errorf("%q is not a valid boolean", val)
		}
	default:
		return nil, errors.New("invalid parameter definition")
	}
}

func (pd ParameterDefinition) validateStringParameterValue(value interface{}) error {
	s, ok := value.(string)
	if !ok {
		return errors.New("value is not a string")
	}
	if pd.MinLength != nil && len(s) < *pd.MinLength {
		return fmt.Errorf("value is too short: minimum length is %d", *pd.MinLength)
	}
	if pd.MaxLength != nil && len(s) > *pd.MaxLength {
		return fmt.Errorf("value is too long: maximum length is %d", *pd.MaxLength)
	}
	return nil
}

func (pd ParameterDefinition) validateIntParameterValue(value interface{}) error {
	i, ok := value.(int)
	if !ok {
		f, ok := value.(float64)
		if !ok {
			return errors.New("value is not a number")
		}
		i, ok = asInt(f)
		if !ok {
			return errors.New("value is not an integer")
		}
	}
	if pd.MinValue != nil && i < *pd.MinValue {
		return fmt.Errorf("value is too low: minimum value is %d", *pd.MinValue)
	}
	if pd.MaxValue != nil && i > *pd.MaxValue {
		return fmt.Errorf("value is too high: maximum value is %d", *pd.MaxValue)
	}
	return nil
}

func (pd ParameterDefinition) validateBoolParameterValue(value interface{}) error {
	_, ok := value.(bool)
	if !ok {
		return errors.New("value is not a boolean")
	}
	return nil
}

func isInCollection(value interface{}, values []interface{}) bool {
	for _, v := range values {
		if value == v {
			return true
		}
	}
	return false
}

func asInt(f float64) (int, bool) {
	i := int(f)
	if float64(i) != f {
		return 0, false
	}
	return i, true
}
