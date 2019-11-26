package internal

import (
	"fmt"
	"testing"

	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
)

func TestAppNameFromDir(t *testing.T) {
	cases := []struct {
		name, expected string
	}{
		{name: "foo", expected: "foo"},
		{name: "foo.dockerapp", expected: "foo"},
		// FIXME(vdemeester) we should fail here
		{name: ".dockerapp", expected: ""},
		{name: "foo/bar", expected: "bar"},
		{name: "foo/bar.dockerapp", expected: "bar"},
		// FIXME(vdemeester) we should fail here
		{name: "foo/bar/.dockerapp", expected: ""},
		{name: "/foo/bar.dockerapp", expected: "bar"},
	}
	for _, tc := range cases {
		assert.Check(t, is.Equal(AppNameFromDir(tc.name), tc.expected))
	}
}

func TestDirNameFromAppName(t *testing.T) {
	cases := []struct {
		name, expected string
	}{
		{name: "foo", expected: "foo.dockerapp"},
		{name: "foo.dockerapp", expected: "foo.dockerapp"},
		// FIXME(vdemeester) we should fail here
		{name: "", expected: ".dockerapp"},
		{name: "foo/bar", expected: "foo/bar.dockerapp"},
		{name: "foo/bar.dockerapp", expected: "foo/bar.dockerapp"},
		{name: "foo/bar.dockerapp/", expected: "foo/bar.dockerapp/"},
		// FIXME(vdemeester) we should fail here
		{name: "foo/bar/", expected: "foo/bar/.dockerapp"},
		{name: "/foo/bar.dockerapp", expected: "/foo/bar.dockerapp"},
	}
	for _, tc := range cases {
		assert.Check(t, is.Equal(DirNameFromAppName(tc.name), tc.expected))
	}
}

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
