package helm

import (
	"testing"

	"gotest.tools/assert"
)

func TestToGoTemplate(t *testing.T) {
	vars := `$VAR ${HELLO} ${WORLD:-world} ${FOOBAR?errbaz} $$`
	result, err := toGoTemplate(vars)
	assert.NilError(t, err)
	assert.Equal(t, result, "{{.Values.VAR}} {{.Values.HELLO}} {{.Values.WORLD}} {{.Values.FOOBAR}} $")
}
