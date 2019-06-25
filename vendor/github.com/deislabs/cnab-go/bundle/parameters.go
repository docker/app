package bundle

type ParametersDefinition struct {
	Fields   map[string]ParameterDefinition `json:"fields" mapstructure:"fields"`
	Required []string                       `json:"required,omitempty" mapstructure:"required,omitempty"`
}

// ParameterDefinition defines a single parameter for a CNAB bundle
type ParameterDefinition struct {
	Definition  string    `json:"definition" mapstructure:"definition"`
	ApplyTo     []string  `json:"applyTo,omitempty" mapstructure:"applyTo,omitempty"`
	Description string    `json:"description,omitempty" mapstructure:"description"`
	Destination *Location `json:"destination,omitemtpty" mapstructure:"destination"`
}
