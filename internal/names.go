package internal

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	// AppExtension is the extension used by an application.
	AppExtension = ".dockerapp"
	// ImageLabel is the label used to distinguish applications from Docker images.
	ImageLabel = "com.docker.application"
	// MetadataFileName is metadata file name
	MetadataFileName = "metadata.yml"
	// ComposeFileName is compose file name
	ComposeFileName = "docker-compose.yml"
	// ParametersFileName is parameters file name
	ParametersFileName = "parameters.yml"
)

var (
	// FileNames lists the application file names, in order.
	FileNames = []string{MetadataFileName, ComposeFileName, ParametersFileName}
)

var appNameRe, _ = regexp.Compile("^[a-zA-Z][a-zA-Z0-9_-]+$")

// AppNameFromDir takes a path to an app directory and returns
// the application's name
func AppNameFromDir(dirName string) string {
	return strings.TrimSuffix(filepath.Base(dirName), AppExtension)
}

// DirNameFromAppName takes an application name and returns the
// corresponding directory name
func DirNameFromAppName(appName string) string {
	if strings.HasSuffix(strings.TrimSuffix(appName, "/"), AppExtension) {
		return appName
	}
	return appName + AppExtension
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
