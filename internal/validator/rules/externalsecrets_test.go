package rules

import (
	"testing"

	"gotest.tools/assert"
)

func TestExternalSecrets(t *testing.T) {
	s := NewExternalSecretsRule()

	t.Run("should accept secrets", func(t *testing.T) {
		// The secrets key is on the root path, that's why it doesn't
		// have a parent
		assert.Equal(t, s.Accept("", "secrets"), true)
	})

	t.Run("should return nil if all secrets are external", func(t *testing.T) {
		input := map[string]interface{}{
			"my_secret": map[string]interface{}{
				"external": "true",
			},
		}

		errs := s.Validate(input)
		assert.Equal(t, len(errs), 0)
	})

	t.Run("should return error if no external secrets", func(t *testing.T) {
		input := map[string]interface{}{
			"my_secret": map[string]interface{}{
				"file": "./my_secret.txt",
			},
		}

		errs := s.Validate(input)
		assert.Equal(t, len(errs), 1)
		assert.ErrorContains(t, errs[0], `secret "my_secret" should be external`)
	})

	t.Run("should return all errors", func(t *testing.T) {
		input := map[string]interface{}{
			"my_secret": map[string]interface{}{
				"file": "./my_secret.txt",
			},
			"my_other_secret": map[string]interface{}{
				"file": "./my_secret.txt",
			},
		}

		errs := s.Validate(input)
		assert.Equal(t, len(errs), 2)
	})

}
