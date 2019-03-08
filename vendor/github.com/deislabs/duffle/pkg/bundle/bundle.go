package bundle

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
)

// Bundle is a CNAB metadata document
type Bundle struct {
	Name             string                         `json:"name" mapstructure:"name"`
	Version          string                         `json:"version" mapstructure:"version"`
	Description      string                         `json:"description" mapstructure:"description"`
	Keywords         []string                       `json:"keywords,omitempty" mapstructure:"keywords,omitempty"`
	Maintainers      []Maintainer                   `json:"maintainers,omitempty" mapstructure:"maintainers,omitempty"`
	InvocationImages []InvocationImage              `json:"invocationImages" mapstructure:"invocationImages"`
	Images           map[string]Image               `json:"images" mapstructure:"images"`
	Actions          map[string]Action              `json:"actions,omitempty" mapstructure:"actions,omitempty"`
	Parameters       map[string]ParameterDefinition `json:"parameters" mapstructure:"parameters"`
	Credentials      map[string]Location            `json:"credentials" mapstructure:"credentials"`
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
	d, err := json.MarshalIndent(b, "", "    ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(dest, d, mode)
}

// WriteTo writes unsigned JSON to an io.Writer using the standard formatting.
func (b Bundle) WriteTo(w io.Writer) (int64, error) {
	d, err := json.MarshalIndent(b, "", "    ")
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
	ImageType string        `json:"imageType" mapstructure:"imageType"`
	Image     string        `json:"image" mapstructure:"image"`
	Digest    string        `json:"digest,omitempty" mapstructure:"digest"`
	Size      uint64        `json:"size,omitempty" mapstructure:"size"`
	Platform  ImagePlatform `json:"platform,omitempty" mapstructure:"platform"`
	MediaType string        `json:"mediaType,omitempty" mapstructure:"mediaType"`
}

// ImagePlatform indicates what type of platform an image is built for
type ImagePlatform struct {
	Architecture string `json:"architecture,omitempty" mapstructure:"architecture"`
	OS           string `json:"os,omitempty" mapstructure:"os"`
}

// Image describes a container image in the bundle
type Image struct {
	BaseImage
	Description string        `json:"description" mapstructure:"description"` //TODO: change? see where it's being used? change to description?
	Refs        []LocationRef `json:"refs" mapstructure:"refs"`
}

// InvocationImage contains the image type and location for the installation of a bundle
type InvocationImage struct {
	BaseImage
}

// Location provides the location where a value should be written in
// the invocation image.
//
// A location may be either a file (by path) or an environment variable.
type Location struct {
	Path                string `json:"path" mapstructure:"path"`
	EnvironmentVariable string `json:"env" mapstructure:"env"`
}

// Maintainer describes a code maintainer of a bundle
type Maintainer struct {
	// Name is a user name or organization name
	Name string `json:"name" mapstructure:"name"`
	// Email is an optional email address to contact the named maintainer
	Email string `json:"email" mapstructure:"email"`
	// Url is an optional URL to an address for the named maintainer
	URL string `json:"url" mapstructure:"url"`
}

// Action describes a custom (non-core) action.
type Action struct {
	// Modifies indicates whether this action modifies the release.
	//
	// If it is possible that an action modify a release, this must be set to true.
	Modifies bool `json:"modifies" mapstructure:"modifies"`
}

// ValuesOrDefaults returns parameter values or the default parameter values
func ValuesOrDefaults(vals map[string]interface{}, b *Bundle) (map[string]interface{}, error) {
	res := map[string]interface{}{}
	for name, def := range b.Parameters {
		if val, ok := vals[name]; ok {
			if err := def.ValidateParameterValue(val); err != nil {
				return res, fmt.Errorf("can't use %v as value of %s: %s", val, name, err)
			}
			typedVal := def.CoerceValue(val)
			res[name] = typedVal
			continue
		} else if def.Required {
			return res, fmt.Errorf("parameter %q is required", name)
		}
		res[name] = def.DefaultValue
	}
	return res, nil
}

// Validate the bundle contents.
func (b Bundle) Validate() error {
	if len(b.InvocationImages) == 0 {
		return errors.New("at least one invocation image must be defined in the bundle")
	}

	if b.Version == "latest" {
		return errors.New("'latest' is not a valid tag option")
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
