package compose

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/docker/cli/cli/compose/loader"
	"github.com/docker/cli/cli/compose/template"
	composetypes "github.com/docker/cli/cli/compose/types"
	"github.com/pkg/errors"
)

const (
	delimiter    = "\\$"
	substitution = "[_a-z][._a-z0-9]*(?::?[-?][^}]*)?"
)

var (
	patternString = fmt.Sprintf(
		"%s(?i:(?P<escaped>%s)|(?P<named>%s)|{(?P<braced>%s)}|(?P<invalid>))",
		delimiter, delimiter, substitution, substitution,
	)
	// Pattern is the variable regexp pattern used to interpolate or extract variables when rendering
	Pattern = regexp.MustCompile(patternString)
)

// Load applies the specified function when loading a slice of compose data
func Load(composes [][]byte, apply func(string) (string, error)) ([]composetypes.ConfigFile, map[string]string, error) {
	configFiles := []composetypes.ConfigFile{}
	for _, data := range composes {
		s, err := apply(string(data))
		if err != nil {
			return nil, nil, err
		}
		parsed, err := loader.ParseYAML([]byte(s))
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to parse Compose file %s", data)
		}
		configFiles = append(configFiles, composetypes.ConfigFile{Config: parsed})
	}

	images, err := validateImagesInConfigFiles(configFiles)
	if err != nil {
		return nil, nil, err
	}

	return configFiles, images, nil
}

// validateImagesInConfigFiles validates that there is no unsupported variable expensions in service images and returns a map of service name -> image
func validateImagesInConfigFiles(configFiles []composetypes.ConfigFile) (map[string]string, error) {
	var errors []string
	images := map[string]string{}
	for _, configFile := range configFiles {
		services, ok := configFile.Config["services"].(map[string]interface{})
		if !ok {
			continue
		}
		for serviceName, serviceContent := range services {
			serviceMap, ok := serviceContent.(map[string]interface{})
			if !ok {
				continue
			}
			imageName, ok := serviceMap["image"].(string)
			if !ok {
				continue
			}
			images[serviceName] = imageName

			if Pattern.MatchString(imageName) {
				errors = append(errors,
					fmt.Sprintf("%s: variables are not allowed in the service's image field. Found: '%s'",
						serviceName, imageName))
			}
		}
	}

	if len(errors) > 0 {
		return nil, fmt.Errorf("%s", strings.Join(errors, "\n"))
	}

	return images, nil
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
