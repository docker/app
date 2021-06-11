package bundle

type Output struct {
	Definition  string   `json:"definition" yaml:"definition"`
	ApplyTo     []string `json:"applyTo,omitempty" yaml:"applyTo,omitempty"`
	Description string   `json:"description,omitempty" yaml:"description,omitempty"`
	Path        string   `json:"path" yaml:"path"`
}

// AppliesTo returns a boolean value specifying whether or not
// the Output applies to the provided action
func (output *Output) AppliesTo(action string) bool {
	if len(output.ApplyTo) == 0 {
		return true
	}
	for _, act := range output.ApplyTo {
		if action == act {
			return true
		}
	}
	return false
}
