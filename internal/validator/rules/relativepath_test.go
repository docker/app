package rules

import (
	"testing"

	"gotest.tools/assert"
)

func TestRelativePathRule(t *testing.T) {
	s := NewRelativePathRule()

	t.Run("should accept only volume paths", func(t *testing.T) {
		assert.Equal(t, s.Accept("services", "test"), false)
		assert.Equal(t, s.Accept("services.test.volumes", "my_volume"), true)
		assert.Equal(t, s.Accept("services.test", "volumes"), true)
	})

	t.Run("should validate named volume paths", func(t *testing.T) {
		input := map[string]string{
			"toto": "tata",
		}
		errs := s.Validate(input)
		assert.Equal(t, len(errs), 0)
	})

	t.Run("should return error if short syntax volume path is relative", func(t *testing.T) {
		input := []interface{}{
			"./foo:/data",
		}
		errs := s.Validate(input)
		assert.Equal(t, len(errs), 1)

		assert.ErrorContains(t, errs[0], `can't use relative path as volume source ("./foo:/data") in service "test"`)
	})

	t.Run("should return error if the volume definition is invalid", func(t *testing.T) {
		input := []interface{}{
			"foo",
		}
		errs := s.Validate(input)
		assert.Equal(t, len(errs), 1)

		assert.ErrorContains(t, errs[0], `invalid volume definition ("foo") in service "test"`)
	})

	t.Run("should return all volume errors", func(t *testing.T) {
		input := []interface{}{
			"./foo:/data1",
			"./bar:/data2",
		}
		errs := s.Validate(input)
		assert.Equal(t, len(errs), 2)

		assert.ErrorContains(t, errs[0], `can't use relative path as volume source ("./foo:/data1") in service "test"`)
		assert.ErrorContains(t, errs[1], `can't use relative path as volume source ("./bar:/data2") in service "test"`)
	})

	// When a volume is in short syntax, the list of volumes must be strings
	t.Run("shoud return error if volume list is invalid", func(t *testing.T) {
		input := []interface{}{
			1,
		}
		errs := s.Validate(input)
		assert.Equal(t, len(errs), 1)

		assert.ErrorContains(t, errs[0], `invalid volume in service "test"`)
	})

	t.Run("should return error if long syntax volume path is relative", func(t *testing.T) {
		input := map[string]interface{}{
			"source": "./foo",
		}
		errs := s.Validate(input)
		assert.Equal(t, len(errs), 1)

		assert.ErrorContains(t, errs[0], `can't use relative path as volume source ("./foo") in service "test"`)
	})

	t.Run("shoud return error if volume map is invalid", func(t *testing.T) {
		input := map[string]interface{}{}
		errs := s.Validate(input)
		assert.Equal(t, len(errs), 1)

		assert.ErrorContains(t, errs[0], `invalid volume in service "test"`)
	})
}
