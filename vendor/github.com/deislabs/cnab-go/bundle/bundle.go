package bundle

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/Masterminds/semver"
	"github.com/deislabs/cnab-go/bundle/definition"
	"github.com/docker/go/canonical/json"
	pkgErrors "github.com/pkg/errors"
)

// Bundle is a CNAB metadata document
type Bundle struct {
	SchemaVersion      string                 `json:"schemaVersion" mapstructure:"schemaVersion"`
	Name               string                 `json:"name" mapstructure:"name"`
	Version            string                 `json:"version" mapstructure:"version"`
	Description        string                 `json:"description" mapstructure:"description"`
	Keywords           []string               `json:"keywords,omitempty" mapstructure:"keywords"`
	Maintainers        []Maintainer           `json:"maintainers,omitempty" mapstructure:"maintainers"`
	InvocationImages   []InvocationImage      `json:"invocationImages" mapstructure:"invocationImages"`
	Images             map[string]Image       `json:"images,omitempty" mapstructure:"images"`
	Actions            map[string]Action      `json:"actions,omitempty" mapstructure:"actions"`
	Parameters         map[string]Parameter   `json:"parameters,omitempty" mapstructure:"parameters"`
	Credentials        map[string]Credential  `json:"credentials,omitempty" mapstructure:"credentials"`
	Outputs            *OutputsDefinition     `json:"outputs,omitempty" mapstructure:"outputs"`
	Definitions        definition.Definitions `json:"definitions,omitempty" mapstructure:"definitions"`
	License            string                 `json:"license,omitempty" mapstructure:"license"`
	RequiredExtensions []string               `json:"requiredExtensions,omitempty" mapstructure:"requiredExtensions"`

	// Custom extension metadata is a named collection of auxiliary data whose
	// meaning is defined outside of the CNAB specification.
	Custom map[string]interface{} `json:"custom,omitempty" mapstructure:"custom"`
}

//Unmarshal unmarshals a Bundle that was not signed.
func Unmarshal(data []byte) (*Bundle, error) {
	b := &Bundle{}
	return b, json.Unmarshal(data, b)
}

// ParseReader reads CNAB metadata from a JSON string
func ParseReader(r io.Reader) (Bundle, error) {
	b := Bundle{}
	err := json.NewDecoder(r).Decode(&b)
	return b, err
}

// WriteFile serializes the bundle and writes it to a file as JSON.
func (b Bundle) WriteFile(dest string, mode os.FileMode) error {
	// FIXME: The marshal here should exactly match the Marshal in the signature code.
	d, err := json.MarshalCanonical(b)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(dest, d, mode)
}

// WriteTo writes unsigned JSON to an io.Writer using the standard formatting.
func (b Bundle) WriteTo(w io.Writer) (int64, error) {
	d, err := json.MarshalCanonical(b)
	if err != nil {
		return 0, err
	}
	l, err := w.Write(d)
	return int64(l), err
}

// LocationRef specifies a location within the invocation package
type LocationRef struct {
	Path      string `json:"path" mapstructure:"path"`
	Field     string `json:"field" mapstructure:"field"`
	MediaType string `json:"mediaType" mapstructure:"mediaType"`
}

// BaseImage contains fields shared across image types
type BaseImage struct {
	ImageType string            `json:"imageType" mapstructure:"imageType"`
	Image     string            `json:"image" mapstructure:"image"`
	Digest    string            `json:"contentDigest,omitempty" mapstructure:"contentDigest"`
	Size      uint64            `json:"size,omitempty" mapstructure:"size"`
	Labels    map[string]string `json:"labels,omitempty" mapstructure:"labels"`
	MediaType string            `json:"mediaType,omitempty" mapstructure:"mediaType"`
}

// Image describes a container image in the bundle
type Image struct {
	BaseImage   `mapstructure:",squash"`
	Description string `json:"description" mapstructure:"description"` //TODO: change? see where it's being used? change to description?
}

// InvocationImage contains the image type and location for the installation of a bundle
type InvocationImage struct {
	BaseImage `mapstructure:",squash"`
}

// ImageRelocationMap stores the relocated images
// The key is the Image in bundle.json and the value is the new Image
// from the relocated registry
type ImageRelocationMap map[string]string

// Location provides the location where a value should be written in
// the invocation image.
//
// A location may be either a file (by path) or an environment variable.
type Location struct {
	Path                string `json:"path,omitempty" mapstructure:"path"`
	EnvironmentVariable string `json:"env,omitempty" mapstructure:"env"`
}

// Maintainer describes a code maintainer of a bundle
type Maintainer struct {
	// Name is a user name or organization name
	Name string `json:"name" mapstructure:"name"`
	// Email is an optional email address to contact the named maintainer
	Email string `json:"email,omitempty" mapstructure:"email"`
	// Url is an optional URL to an address for the named maintainer
	URL string `json:"url,omitempty" mapstructure:"url"`
}

// Action describes a custom (non-core) action.
type Action struct {
	// Modifies indicates whether this action modifies the release.
	//
	// If it is possible that an action modify a release, this must be set to true.
	Modifies bool `json:"modifies,omitempty" mapstructure:"modifies"`
	// Stateless indicates that the action is purely informational, that credentials are not required, and that the runtime should not keep track of its invocation
	Stateless bool `json:"stateless,omitempty" mapstructure:"stateless"`
	// Description describes the action as a user-readable string
	Description string `json:"description,omitempty" mapstructure:"description"`
}

// ValuesOrDefaults returns parameter values or the default parameter values. An error is returned when the parameter value does not pass
// the schema validation, a required parameter is missing or an immutable parameter is set with a new value.
func ValuesOrDefaults(vals map[string]interface{}, currentVals map[string]interface{}, b *Bundle) (map[string]interface{}, error) {
	res := map[string]interface{}{}

	for name, param := range b.Parameters {
		s, ok := b.Definitions[param.Definition]
		if !ok {
			return res, fmt.Errorf("unable to find definition for %s", name)
		}
		if val, ok := vals[name]; ok {
			if currentVal, ok := currentVals[name]; param.Immutable && ok && currentVal != val {
				return res, fmt.Errorf("parameter %s is immutable and cannot be overridden with value %v", name, val)
			}
			valErrs, err := s.Validate(val)
			if err != nil {
				return res, pkgErrors.Wrapf(err, "encountered an error validating parameter %s", name)
			}
			// This interface returns a single error. Validation can have multiple errors. For now return the first
			// We should update this later.
			if len(valErrs) > 0 {
				valErr := valErrs[0]
				return res, fmt.Errorf("cannot use value: %v as parameter %s: %s ", val, name, valErr.Error)
			}
			typedVal := s.CoerceValue(val)
			res[name] = typedVal
			continue
		} else if param.Required {
			return res, fmt.Errorf("parameter %q is required", name)
		}
		res[name] = s.Default
	}
	return res, nil
}

// Validate the bundle contents.
func (b Bundle) Validate() error {
	_, err := semver.NewVersion(b.SchemaVersion)
	if err != nil {
		return fmt.Errorf("invalid bundle schema version %q: %v", b.SchemaVersion, err)
	}

	if len(b.InvocationImages) == 0 {
		return errors.New("at least one invocation image must be defined in the bundle")
	}

	if b.Version == "latest" {
		return errors.New("'latest' is not a valid bundle version")
	}

	reqExt := make(map[string]bool, len(b.RequiredExtensions))
	for _, requiredExtension := range b.RequiredExtensions {
		// Verify the custom extension declared as required exists
		if _, exists := b.Custom[requiredExtension]; !exists {
			return fmt.Errorf("required extension '%s' is not defined in the Custom section of the bundle", requiredExtension)
		}

		// Check for duplicate entries
		if _, exists := reqExt[requiredExtension]; exists {
			return fmt.Errorf("required extension '%s' is already declared", requiredExtension)
		}

		// Populate map with required extension, for duplicate check above
		reqExt[requiredExtension] = true
	}

	for _, img := range b.InvocationImages {
		err := img.Validate()
		if err != nil {
			return err
		}
	}

	return nil
}

// Validate the image contents.
func (img InvocationImage) Validate() error {
	switch img.ImageType {
	case "docker", "oci":
		return validateDockerish(img.Image)
	default:
		return nil
	}
}

func validateDockerish(s string) error {
	if !strings.Contains(s, ":") {
		return errors.New("tag is required")
	}
	return nil
}
