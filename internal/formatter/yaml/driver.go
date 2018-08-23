package yaml

import (
    "github.com/docker/app/internal/formatter"
    "github.com/docker/app/internal/yaml"
    composetypes "github.com/docker/cli/cli/compose/types"
    "github.com/pkg/errors"
)

func init() {
    formatter.Register("yaml", &Driver{})
}

// Driver is the yaml implementation of formatter drivers.
type Driver struct{}

// Format creates a YAML document from the source config.
func (d *Driver) Format(config *composetypes.Config) (string, error) {
    result, err := yaml.Marshal(config)
    if err != nil {
        return "", errors.Wrap(err, "failed to produce yaml structure")
    }
    return string(result), nil
}
