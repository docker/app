package driver

import (
	"fmt"
	"io"

	"github.com/docker/go/canonical/json"
)

// ImageType constants provide some of the image types supported
// TODO: I think we can remove all but Docker, since the rest are supported externally
const (
	ImageTypeDocker = "docker"
	ImageTypeOCI    = "oci"
	ImageTypeQCOW   = "qcow"
)

// Operation describes the data passed into the driver to run an operation
type Operation struct {
	// Installation is the name of this installation
	Installation string `json:"installation_name"`
	// The revision ID for this installation
	Revision string `json:"revision"`
	// Action is the action to be performed
	Action string `json:"action"`
	// Parameters are the parameters to be injected into the container
	Parameters map[string]interface{} `json:"parameters"`
	// Image is the invocation image
	Image string `json:"image"`
	// ImageType is the type of image.
	ImageType string `json:"image_type"`
	// Environment contains environment variables that should be injected into the invocation image
	Environment map[string]string `json:"environment"`
	// Files contains files that should be injected into the invocation image.
	Files map[string]string `json:"files"`
	// Output stream for log messages from the driver
	Out io.Writer `json:"-"`
}

// ResolvedCred is a credential that has been resolved and is ready for injection into the runtime.
type ResolvedCred struct {
	Type  string `json:"type"`
	Name  string `json:"name"`
	Value string `json:"value"`
}

// Driver is capable of running a invocation image
type Driver interface {
	// Run executes the operation inside of the invocation image
	Run(*Operation) error
	// Handles receives an ImageType* and answers whether this driver supports that type
	Handles(string) bool
}

// Configurable drivers can explain their configuration, and have it explicitly set
type Configurable interface {
	// Config returns a map of configuration names and values that can be set via environment variable
	Config() map[string]string
	// SetConfig allows setting configuration, where name corresponds to the key in Config, and value is
	// the value to be set.
	SetConfig(map[string]string)
}

// DebugDriver prints the information passed to a driver
//
// It does not ever run the image.
type DebugDriver struct {
	config map[string]string
}

// Run executes the operation on the Debug driver
func (d *DebugDriver) Run(op *Operation) error {
	data, err := json.MarshalIndent(op, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintln(op.Out, string(data))
	return nil
}

// Handles always returns true, effectively claiming to work for any image type
func (d *DebugDriver) Handles(dt string) bool {
	return true
}

// Config returns the configuration help text
func (d *DebugDriver) Config() map[string]string {
	return map[string]string{
		"VERBOSE": "Increase verbosity. true, false are supported values",
	}
}

// SetConfig sets configuration for this driver
func (d *DebugDriver) SetConfig(settings map[string]string) {
	d.config = settings
}
