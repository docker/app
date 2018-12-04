package renderer

import (
	"testing"

	"github.com/docker/app/internal/renderer/driver"
	"github.com/pkg/errors"
	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
)

type fakeDriver struct{}

func (d *fakeDriver) Apply(s string, parameters map[string]interface{}) (string, error) {
	return s + "fake", nil
}

type fakeErrorDriver struct{}

func (d *fakeErrorDriver) Apply(s string, parameters map[string]interface{}) (string, error) {
	return s, errors.New("error in driver")
}

func TestRegisterNilPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Errorf("The code did not panic")
		}
		resetDrivers()
	}()
	Register("foo", nil)
}

func TestRegisterDuplicatePanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Errorf("The code did not panic")
		}
		resetDrivers()
	}()
	Register("bar", &fakeDriver{})
	Register("bar", &fakeDriver{})
}

func TestRegister(t *testing.T) {
	d := &fakeDriver{}
	Register("baz", d)
	defer resetDrivers()
	assert.Check(t, is.DeepEqual(drivers, map[string]driver.Driver{"baz": d}))
}

func TestNoDrivers(t *testing.T) {
	assert.Check(t, is.DeepEqual(Drivers(), []string{"none"}))
}

func TestRegisteredDrivers(t *testing.T) {
	Register("foo", &fakeDriver{})
	Register("bar", &fakeDriver{})
	defer resetDrivers()
	assert.Check(t, is.DeepEqual(Drivers(), []string{"bar", "foo", "none"}))
}

func TestApplyNonExistentDriver(t *testing.T) {
	_, err := Apply("foo", nil, "toto")
	assert.Check(t, err != nil)
	assert.Check(t, is.ErrorContains(err, "unknown renderer toto"))
}

func TestApplyErrorDriver(t *testing.T) {
	Register("err", &fakeErrorDriver{})
	defer resetDrivers()
	_, err := Apply("foo", nil, "err")
	assert.Check(t, err != nil)
	assert.Check(t, is.ErrorContains(err, "error in driver"))
}

func TestApplyNone(t *testing.T) {
	Register("fake", &fakeDriver{})
	defer resetDrivers()
	s, err := Apply("foo", nil, "none")
	assert.NilError(t, err)
	assert.Check(t, is.Equal(s, "foo"))
}

func TestApply(t *testing.T) {
	Register("fake", &fakeDriver{})
	defer resetDrivers()
	s, err := Apply("foo", nil, "fake")
	assert.NilError(t, err)
	assert.Check(t, is.Equal(s, "foofake"))
}

func resetDrivers() {
	drivers = map[string]driver.Driver{}
}
