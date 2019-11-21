package rules

import (
	"github.com/pkg/errors"
)

type externalSecretsValidator struct {
}

func NewExternalSecretsRule() Rule {
	return &externalSecretsValidator{}
}

func (s *externalSecretsValidator) Collect(parent string, key string, value interface{}) {

}

func (s *externalSecretsValidator) Accept(parent string, key string) bool {
	return key == "secrets"
}

func (s *externalSecretsValidator) Validate(cfgMap interface{}) []error {
	errs := []error{}
	if value, ok := cfgMap.(map[string]interface{}); ok {
		for secretName, secret := range value {
			if v1, ok := secret.(map[string]interface{}); ok {
				var hasExternal = false
				for key := range v1 {
					if key == "external" {
						hasExternal = true
					}
				}
				if !hasExternal {
					errs = append(errs, errors.Errorf(`secret %q should be external`, secretName))
				}
			}
		}
	}
	return errs
}
