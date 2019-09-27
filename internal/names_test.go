package internal

import (
	"fmt"
	"testing"

	"gotest.tools/assert"
)


func TestValidateAppName(t *testing.T) {
	validNames := []string{
		"app", "app1", "my-app", "their_app", "app_01_02_-",
		"LunchBox", "aPP01", "APP365",
	}

	invalidNames := []string{
		"_app", "-app", "01_app", "$$$$$", "app$", "/my/app",
		"(u|\\|[|-||30><", "Our Fortress Is Burning", "d\nx",
		"my_\"app\"",
	}

	for _, name := range validNames {
		err := ValidateAppName(name)
		assert.NilError(t, err)
	}

	for _, name := range invalidNames {
		err := ValidateAppName(name)
		assert.ErrorContains(t, err, fmt.Sprintf("invalid app name: %s", name))
	}
}
