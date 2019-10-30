package bundle

// Parameter defines a single parameter for a CNAB bundle
type Parameter struct {
	Definition  string    `json:"definition" yaml:"definition"`
	ApplyTo     []string  `json:"applyTo,omitempty" yaml:"applyTo,omitempty"`
	Description string    `json:"description,omitempty" yaml:"description,omitempty"`
	Destination *Location `json:"destination,omitemtpty" yaml:"destination,omitempty"`
	Required    bool      `json:"required,omitempty" yaml:"required,omitempty"`
}

// AppliesTo returns a boolean value specifying whether or not
// the Parameter applies to the provided action
func (parameter *Parameter) AppliesTo(action string) bool {
	if len(parameter.ApplyTo) == 0 {
		return true
	}
	for _, act := range parameter.ApplyTo {
		if action == act {
			return true
		}
	}
	return false
}
