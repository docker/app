package validator

import (
	"errors"
	"fmt"
	"strings"

	"github.com/docker/app/internal"
	"github.com/docker/app/render"
	"github.com/docker/app/specification"
	"github.com/docker/app/types"
	"github.com/docker/cli/cli/compose/loader"
)

// Validate checks an application definition meets the specifications (metadata and rendered compose file)
func Validate(app *types.App, env map[string]string) error {
	var errs []string
	if err := validateMetadata(app.Metadata()); err != nil {
		errs = append(errs, err.Error())
	}
	if _, err := render.Render(app, env); err != nil {
		errs = append(errs, err.Error())
	}
	return concatenateErrors(errs)
}

func validateMetadata(metadata []byte) error {
	metadataYaml, err := loader.ParseYAML(metadata)
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
