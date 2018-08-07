package compose

import (
	"regexp"

	"github.com/docker/cli/cli/compose/loader"
	"github.com/docker/cli/cli/compose/template"
	composetypes "github.com/docker/cli/cli/compose/types"
	"github.com/pkg/errors"
)

// Load applies the specified function when loading a slice of compose data
func Load(composes [][]byte, apply func(string) (string, error)) ([]composetypes.ConfigFile, error) {
	configFiles := []composetypes.ConfigFile{}
	for _, data := range composes {
		s, err := apply(string(data))
		if err != nil {
			return nil, err
		}
		parsed, err := loader.ParseYAML([]byte(s))
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse Compose file %s", data)
		}
		configFiles = append(configFiles, composetypes.ConfigFile{Config: parsed})
	}
	return configFiles, nil
}

// ExtractVariables extracts the variables from the specified compose data
// This is a small helper to docker/cli template.ExtractVariables function
func ExtractVariables(data []byte, pattern *regexp.Regexp) (map[string]string, error) {
	cfgMap, err := loader.ParseYAML(data)
	if err != nil {
		return nil, err
	}
	return template.ExtractVariables(cfgMap, pattern), nil
}
