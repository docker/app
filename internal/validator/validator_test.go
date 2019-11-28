package validator

import (
	"testing"

	"github.com/docker/app/internal"
	"gotest.tools/assert"
	"gotest.tools/fs"
)

type mockRule struct {
	acceptCalled   bool
	validateCalled bool
}

func (m *mockRule) Collect(path string, key string, value interface{}) {

}

func (m *mockRule) Accept(path string, key string) bool {
	m.acceptCalled = true
	return true
}

func (m *mockRule) Validate(value interface{}) []error {
	m.validateCalled = true
	return nil
}

func TestValidate(t *testing.T) {
	composeData := `
version: '3.7'
services:
  nginx:
    image: nginx
    volumes:
      - ./foo:/data
`
	inputDir := fs.NewDir(t, "app_input_",
		fs.WithFile(internal.ComposeFileName, composeData),
	)
	defer inputDir.Remove()

	appName := "my.dockerapp"
	dir := fs.NewDir(t, "app_",
		fs.WithDir(appName),
	)
	defer dir.Remove()

	r := &mockRule{}
	v := NewValidator(func(v *Validator) {
		v.Rules = append(v.Rules, r)
	})

	err := v.Validate(inputDir.Join(internal.ComposeFileName))
	assert.NilError(t, err)
	assert.Equal(t, r.acceptCalled, true)
	assert.Equal(t, r.validateCalled, true)
}
