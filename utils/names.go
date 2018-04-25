package utils

import (
	"fmt"
	"path"
	"regexp"
	"strings"

	"github.com/docker/lunchbox/constants"
)

var appNameRe, _ = regexp.Compile("^[a-zA-Z][a-zA-Z0-9_-]+$")

// AppNameFromDir takes a path to an app directory and returns
// the application's name
func AppNameFromDir(dirName string) string {
	return strings.TrimSuffix(path.Base(dirName), constants.AppExtension)
}

// DirNameFromAppName takes an application name and returns the
// corresponding directory name
func DirNameFromAppName(appName string) string {
	return fmt.Sprintf("%s%s", appName, constants.AppExtension)
}

// ValidateAppName takes an app name and returns an error if it doesn't
// match the expected format
func ValidateAppName(appName string) error {
	if appNameRe.MatchString(appName) {
		return nil
	}
	return fmt.Errorf(
		"invalid app name: %s ; app names must start with a letter, and must contain only letters, numbers, '-' and '_' (regexp: %q)",
		appName,
		appNameRe.String(),
	)
}
