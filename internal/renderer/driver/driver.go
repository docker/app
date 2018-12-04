package driver

// Driver is the interface that must be implemented by a renderer driver.
type Driver interface {
	// Apply applies the parameters to the string
	Apply(s string, parameters map[string]interface{}) (string, error)
}
