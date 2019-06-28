package credentials

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"github.com/deislabs/cnab-go/bundle"

	yaml "gopkg.in/yaml.v2"
)

// Set is an actual set of resolved credentials.
// This is the output of resolving a credentialset file.
type Set map[string]string

// Expand expands the set into env vars and paths per the spec in the bundle.
//
// This matches the credentials required by the bundle to the credentials present
// in the credentialset, and then expands them per the definition in the Bundle.
func (s Set) Expand(b *bundle.Bundle, stateless bool) (env, files map[string]string, err error) {
	env, files = map[string]string{}, map[string]string{}
	for name, val := range b.Credentials {
		src, ok := s[name]
		if !ok {
			if stateless {
				continue
			}
			err = fmt.Errorf("credential %q is missing from the user-supplied credentials", name)
			return
		}
		if val.EnvironmentVariable != "" {
			env[val.EnvironmentVariable] = src
		}
		if val.Path != "" {
			files[val.Path] = src
		}
	}
	return
}

// Merge merges a second Set into the base.
//
// Duplicate credential names are not allow and will result in an
// error, this is the case even if the values are identical.
func (s Set) Merge(s2 Set) error {
	for k, v := range s2 {
		if _, ok := s[k]; ok {
			return fmt.Errorf("ambiguous credential resolution: %q is already present in base credential sets, cannot merge", k)
		}
		s[k] = v
	}
	return nil
}

// CredentialSet represents a collection of credentials
type CredentialSet struct {
	// Name is the name of the credentialset.
	Name string `json:"name" yaml:"name"`
	// Creadentials is a list of credential specs.
	Credentials []CredentialStrategy `json:"credentials" yaml:"credentials"`
}

// Load a CredentialSet from a file at a given path.
//
// It does not load the individual credentials.
func Load(path string) (*CredentialSet, error) {
	cset := &CredentialSet{}
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return cset, err
	}
	return cset, yaml.Unmarshal(data, cset)
}

// Validate compares the given credentials with the spec.
//
// This will result in an error only if:
// - a parameter in the spec is not present in the given set
// - a parameter in the given set does not match the format required by the spec
//
// It is allowed for spec to specify both an env var and a file. In such case, if
// the givn set provides either, it will be considered valid.
func Validate(given Set, spec map[string]bundle.Credential) error {
	for name := range spec {
		if !isValidCred(given, name) {
			return fmt.Errorf("bundle requires credential for %s", name)
		}
	}
	return nil
}

func isValidCred(haystack Set, needle string) bool {
	for name := range haystack {
		if name == needle {
			return true
		}
	}
	return false
}

// Resolve looks up the credentials as described in Source, then copies
// the resulting value into the Value field of each credential strategy.
//
// The typical workflow for working with a credential set is:
//
//	- Load the set
//	- Validate the credentials against a spec
//	- Resolve the credentials
//	- Expand them into bundle values
func (c *CredentialSet) Resolve() (Set, error) {
	l := len(c.Credentials)
	res := make(map[string]string, l)
	for i := 0; i < l; i++ {
		cred := c.Credentials[i]
		src := cred.Source
		// Precedence is Command, Path, EnvVar, Value
		switch {
		case src.Command != "":
			data, err := execCmd(src.Command)
			if err != nil {
				return res, err
			}
			cred.Value = string(data)
		case src.Path != "":
			data, err := ioutil.ReadFile(os.ExpandEnv(src.Path))
			if err != nil {
				return res, fmt.Errorf("credential %q: %s", c.Credentials[i].Name, err)
			}
			cred.Value = string(data)
		case src.EnvVar != "":
			var ok bool
			cred.Value, ok = os.LookupEnv(src.EnvVar)
			if ok {
				break
			}
			fallthrough
		default:
			cred.Value = src.Value
		}
		res[c.Credentials[i].Name] = cred.Value
	}
	return res, nil
}

func execCmd(cmd string) ([]byte, error) {
	parts := strings.Split(cmd, " ")
	c := parts[0]
	args := parts[1:]
	run := exec.Command(c, args...)

	return run.CombinedOutput()
}

// CredentialStrategy represents a source credential and the destination to which it should be sent.
type CredentialStrategy struct {
	// Name is the name of the credential.
	// Name is used to match a credential strategy to a bundle's credential.
	Name string `json:"name" yaml:"name"`
	// Source is the location of the credential.
	// During resolution, the source will be loaded, and the result temporarily placed
	// into Value.
	Source Source `json:"source,omitempty" yaml:"source,omitempty"`
	// Value holds the credential value.
	// When a credential is loaded, it is loaded into this field. In all
	// other cases, it is empty. This field is omitted during serialization.
	Value string `json:"-" yaml:"-"`
}

// Source represents a strategy for loading a credential from local host.
type Source struct {
	Path    string `json:"path,omitempty" yaml:"path,omitempty"`
	Command string `json:"command,omitempty" yaml:"command,omitempty"`
	Value   string `json:"value,omitempty" yaml:"value,omitempty"`
	EnvVar  string `json:"env,omitempty" yaml:"env,omitempty"`
}

// Destination reprents a strategy for injecting a credential into an image.
type Destination struct {
	Value string `json:"value,omitempty" yaml:"value,omitempty"`
}
