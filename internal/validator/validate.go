package validator

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/docker/app/internal"
	"github.com/docker/app/internal/render"
	"github.com/docker/app/specification"
	"github.com/docker/cli/cli/compose/loader"
)

// Validate checks an application definition meets the specifications (metadata and rendered compose file)
func Validate(appname string, settingsFiles []string, env map[string]string) error {
	if err := checkExistingFiles(appname); err != nil {
		return err
	}
	if err := validateMetadata(appname); err != nil {
		return err
	}
	if _, err := render.Render(appname, nil, settingsFiles, env); err != nil {
		return err
	}
	return nil
}

func checkExistingFiles(appname string) error {
	if _, err := os.Stat(filepath.Join(appname, internal.SettingsFileName)); err != nil {
		return errors.New("failed to read application settings")
	}
	if _, err := os.Stat(filepath.Join(appname, internal.MetadataFileName)); err != nil {
		return errors.New("failed to read application metadata")
	}
	if _, err := os.Stat(filepath.Join(appname, internal.ComposeFileName)); err != nil {
		return errors.New("failed to read application compose")
	}
	return nil
}

func validateMetadata(appname string) error {
	metadata, err := ioutil.ReadFile(filepath.Join(appname, internal.MetadataFileName))
	if err != nil {
		return fmt.Errorf("failed to read application metadata: %s", err)
	}
	metadataYaml, err := loader.ParseYAML(metadata)
	if err != nil {
		return fmt.Errorf("failed to parse application metadata: %s", err)
	}
	if err := specification.Validate(metadataYaml, internal.MetadataVersion); err != nil {
		return fmt.Errorf("failed to validate metadata:\n%s", err)
	}
	return nil
}
