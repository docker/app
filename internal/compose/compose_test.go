package compose

import (
	"testing"

	"github.com/docker/app/internal/renderer"
	"gotest.tools/assert"

	composetypes "github.com/docker/cli/cli/compose/types"
)

const (
	imageVarsValidationErrorMessage = "variables are not allowed in the service's image field. Found: "
)

func TestValidateNonDynamicImageField(t *testing.T) {
	composeFile := []byte(`
version: "3.6"
services:
    hello:
        image: ${image}`)
	_, err := runLoad(composeFile)
	assert.ErrorContains(t, err, imageVarsValidationErrorMessage)
}

func TestValidateNonDynamicImageFieldNoBrackets(t *testing.T) {
	composeFile := []byte(`
version: "3.6"
services:
    hello:
        image: $image`)
	_, err := runLoad(composeFile)
	assert.ErrorContains(t, err, imageVarsValidationErrorMessage)
}

func TestValidateNonDynamicImageFieldPartial(t *testing.T) {
	composeFile := []byte(`
version: "3.6"
services:
    hello:
        image: prefix-${image}:v1`)
	_, err := runLoad(composeFile)
	assert.ErrorContains(t, err, imageVarsValidationErrorMessage)
}

func TestValidateNonDynamicImageFieldPartialNoBrackets(t *testing.T) {
	composeFile := []byte(`
version: "3.6"
services:
    hello:
        image: prefix-$image:v1`)
	_, err := runLoad(composeFile)
	assert.ErrorContains(t, err, imageVarsValidationErrorMessage)
}

func runLoad(composeFile []byte) ([]composetypes.ConfigFile, error) {
	return Load([][]byte{composeFile}, func(data string) (string, error) {
		return renderer.Apply(data, map[string]interface{}{"image": "nginx"})
	})
}
