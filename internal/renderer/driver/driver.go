package driver

// Driver is the interface that must be implemented by a renderer driver.
type Driver interface {
	// Apply applies the settings to the string
	Apply(s string, settings map[string]interface{}) (string, error)
}
