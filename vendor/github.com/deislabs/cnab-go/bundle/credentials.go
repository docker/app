package bundle

// Credential represents the definition of a CNAB credential
type Credential struct {
	Location    `mapstructure:",squash"`
	Description string `json:"description,omitempty" mapstructure:"description"`
	Required    bool   `json:"required,omitempty" mapstructure:"required"`
}
