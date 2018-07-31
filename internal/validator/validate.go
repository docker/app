package validator

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/app/internal"
	"github.com/docker/app/internal/render"
	"github.com/docker/app/internal/templateloader"
	"github.com/docker/app/specification"
)

// Validate checks an application definition meets the specifications (metadata and rendered compose file)
func Validate(appname string, settingsFiles []string, env map[string]string) error {
	var errs []string
	if err := checkExistingFiles(appname); err != nil {
		errs = append(errs, err.Error())
	}
	if err := validateMetadata(appname); err != nil {
		errs = append(errs, err.Error())
	}
	if _, err := render.Render(appname, nil, settingsFiles, env); err != nil {
		errs = append(errs, err.Error())
	}
	return concatenateErrors(errs)
}

func checkExistingFiles(appname string) error {
	var errs []string
	if _, err := os.Stat(filepath.Join(appname, internal.SettingsFileName)); err != nil {
		errs = append(errs, "failed to read application settings")
	}
	if _, err := os.Stat(filepath.Join(appname, internal.MetadataFileName)); err != nil {
		errs = append(errs, "failed to read application metadata")
	}
	if _, err := os.Stat(filepath.Join(appname, internal.ComposeFileName)); err != nil {
		errs = append(errs, "failed to read application compose")
	}
	return concatenateErrors(errs)
}

func validateMetadata(appname string) error {
	metadata, err := ioutil.ReadFile(filepath.Join(appname, internal.MetadataFileName))
	if err != nil {
		return fmt.Errorf("failed to read application metadata: %s", err)
	}
	metadataYaml, err := templateloader.ParseYAML(metadata)
	if err != nil {
		return fmt.Errorf("failed to parse application metadata: %s", err)
	}
	if err := specification.Validate(metadataYaml, internal.MetadataVersion); err != nil {
		return fmt.Errorf("failed to validate metadata:\n%s", err)
	}
	return nil
}

func concatenateErrors(errs []string) error {
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "\n"))
	}
	return nil
}
