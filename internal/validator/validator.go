package validator

import (
	"io/ioutil"
	"sort"
	"strings"

	"github.com/docker/app/internal/validator/rules"
	composeloader "github.com/docker/cli/cli/compose/loader"
	"github.com/pkg/errors"
)

type Validator struct {
	Rules  []rules.Rule
	errors []error
}

type ValidationError struct {
	Errors []error
}

type ValidationCallback func(string, string, interface{})

func (v ValidationError) Error() string {
	parts := []string{}
	for _, err := range v.Errors {
		parts = append(parts, "* "+err.Error())
	}

	sort.Strings(parts)
	parts = append([]string{"Compose file validation failed:"}, parts...)

	return strings.Join(parts, "\n")
}

type Config func(*Validator)
type Opt func(c *Validator) error

func NewValidator(opts ...Config) Validator {
	validator := Validator{}
	for _, opt := range opts {
		opt(&validator)
	}
	return validator
}

func WithRelativePathRule() Config {
	return func(v *Validator) {
		v.Rules = append(v.Rules, rules.NewRelativePathRule())
	}
}

func WithExternalSecretsRule() Config {
	return func(v *Validator) {
		v.Rules = append(v.Rules, rules.NewExternalSecretsRule())
	}
}

func NewValidatorWithDefaults() Validator {
	return NewValidator(
		WithRelativePathRule(),
		WithExternalSecretsRule(),
	)
}

// Validate validates the compose file, it returns an error
// if it can't parse the compose file or a ValidationError
// that contains all the validation errors (if any), nil otherwise
func (v *Validator) Validate(composeFile string) error {
	composeRaw, err := ioutil.ReadFile(composeFile)
	if err != nil {
		return errors.Wrapf(err, "failed to read compose file %q", composeFile)
	}
	cfgMap, err := composeloader.ParseYAML(composeRaw)
	if err != nil {
		return errors.Wrap(err, "failed to parse compose file")
	}

	// First phase, the rules collect all the dependent values they need
	v.visitAll("", cfgMap, v.collect)
	// Second phase, validate the compose file
	v.visitAll("", cfgMap, v.validate)

	if len(v.errors) > 0 {
		return ValidationError{
			Errors: v.errors,
		}
	}
	return nil
}

func (v *Validator) collect(parent string, key string, value interface{}) {
	for _, rule := range v.Rules {
		rule.Collect(parent, key, value)
	}
}

func (v *Validator) validate(parent string, key string, value interface{}) {
	for _, rule := range v.Rules {
		if rule.Accept(parent, key) {
			verrs := rule.Validate(value)
			if len(verrs) > 0 {
				v.errors = append(v.errors, verrs...)
			}
		}
	}
}

func (v *Validator) visitAll(parent string, cfgMap interface{}, cb ValidationCallback) {
	m, ok := cfgMap.(map[string]interface{})
	if !ok {
		return
	}

	for key, value := range m {
		switch value := value.(type) {
		case string:
			continue
		default:
			cb(parent, key, value)

			path := parent + "." + key
			if parent == "" {
				path = key
			}

			sub, ok := m[key].(map[string]interface{})
			if ok {
				v.visitAll(path, sub, cb)
			}
		}
	}
}
