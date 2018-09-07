package json

import (
	"encoding/json"

	"github.com/docker/app/internal/formatter"
	composetypes "github.com/docker/cli/cli/compose/types"
	"github.com/pkg/errors"
)

func init() {
	formatter.Register("json", &Driver{})
}

// Driver is the json implementation of formatter drivers.
type Driver struct{}

// Format creates a JSON document from the source config.
func (d *Driver) Format(config *composetypes.Config) (string, error) {
	result, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return "", errors.Wrap(err, "failed to produce json structure")
	}
	return string(result) + "\n", nil
}
