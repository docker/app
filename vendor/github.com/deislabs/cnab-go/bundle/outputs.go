package bundle

type OutputsDefinition struct {
	Fields map[string]OutputDefinition `json:"fields" mapstructure:"fields"`
}

type OutputDefinition struct {
	Definition  string   `json:"definition" mapstructure:"definition"`
	ApplyTo     []string `json:"applyTo,omitempty" mapstructure:"applyTo,omitempty"`
	Description string   `json:"description,omitempty" mapstructure:"description"`
	Path        string   `json:"path" mapstructure:"path"`
}
