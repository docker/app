package metadata

import (
	"fmt"

	"github.com/docker/app/internal"
	"github.com/docker/app/internal/yaml"
	"github.com/docker/app/specification"
	"github.com/docker/cli/cli/compose/loader"
	"github.com/pkg/errors"
)

// Load validates the given data and loads it into a metadata struct
func Load(data []byte) (AppMetadata, error) {
	if err := validateRawMetadata(data); err != nil {
		return AppMetadata{}, err
	}
	var meta AppMetadata
	if err := yaml.Unmarshal(data, &meta); err != nil {
		return AppMetadata{}, errors.Wrap(err, "failed to unmarshal metadata")
	}
	return meta, nil
}

func validateRawMetadata(metadata []byte) error {
	metadataYaml, err := loader.ParseYAML(metadata)
	if err != nil {
		return fmt.Errorf("failed to parse application metadata: %s", err)
	}
	if err := specification.Validate(metadataYaml, internal.MetadataVersion); err != nil {
		return fmt.Errorf("failed to validate metadata:\n%s", err)
	}
	return nil
}
