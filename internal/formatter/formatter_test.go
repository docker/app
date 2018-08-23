package formatter

import (
    "testing"

    "github.com/docker/app/internal/formatter/driver"
    composetypes "github.com/docker/cli/cli/compose/types"
    "github.com/pkg/errors"
    "gotest.tools/assert"
    is "gotest.tools/assert/cmp"
)

type fakeDriver struct{}

func (d *fakeDriver) Format(config *composetypes.Config) (string, error) {
    return "fake", nil
}

type fakeErrorDriver struct{}

func (d *fakeErrorDriver) Format(config *composetypes.Config) (string, error) {
    return "", errors.New("error in driver")
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
    assert.Check(t, is.DeepEqual(Drivers(), []string{}))
}

func TestRegisteredDrivers(t *testing.T) {
    Register("foo", &fakeDriver{})
    Register("bar", &fakeDriver{})
    defer resetDrivers()
    assert.Check(t, is.DeepEqual(Drivers(), []string{"bar", "foo"}))
}

func TestFormatNonExistentDriver(t *testing.T) {
    _, err := Format(&composetypes.Config{}, "toto")
    assert.Check(t, err != nil)
    assert.Check(t, is.ErrorContains(err, "unknown formatter toto"))
}

func TestFormatErrorDriver(t *testing.T) {
    Register("err", &fakeErrorDriver{})
    defer resetDrivers()
    _, err := Format(&composetypes.Config{}, "err")
    assert.Check(t, err != nil)
    assert.Check(t, is.ErrorContains(err, "error in driver"))
}

func TestFormatNone(t *testing.T) {
    Register("fake", &fakeDriver{})
    defer resetDrivers()
    _, err := Format(&composetypes.Config{}, "none")
    assert.Check(t, err != nil)
    assert.Check(t, is.ErrorContains(err, "unknown formatter none"))
}

func TestFormat(t *testing.T) {
    Register("fake", &fakeDriver{})
    defer resetDrivers()
    s, err := Format(&composetypes.Config{}, "fake")
    assert.NilError(t, err)
    assert.Check(t, is.Equal(s, "fake"))
}

func resetDrivers() {
    drivers = map[string]driver.Driver{}
}
