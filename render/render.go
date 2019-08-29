package render

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/deislabs/cnab-go/bundle"
	"github.com/docker/app/internal/compose"
	"github.com/docker/app/types"
	"github.com/docker/app/types/parameters"
	"github.com/docker/cli/cli/compose/loader"
	composetypes "github.com/docker/cli/cli/compose/types"
	"github.com/pkg/errors"

	// Register json formatter
	_ "github.com/docker/app/internal/formatter/json"
	// Register yaml formatter
	_ "github.com/docker/app/internal/formatter/yaml"
)

// pattern matching for ${text} and $text substrings (characters allowed: 0-9 a-z _ .)
const (
	delimiter           = `\$`
	substitution        = `[a-zA-Z_]+([a-zA-Z0-9_]*(([.]{1}[0-9a-zA-Z_]+)|([0-9a-zA-Z_])))*`
	defaultValuePattern = `[a-zA-Z_]+[a-zA-Z0-9_.]*((:-)|(\-)|(:\?)|(\?))(.*)`
)

var (
	patternString = fmt.Sprintf(
		`%s(?i:(?P<named>%s)|(?P<skip>%s{1,})|\{(?P<braced>%s)\}|\{(?P<fail>%s)\})`,
		delimiter, substitution, delimiter, substitution, defaultValuePattern,
	)
	rePattern = regexp.MustCompile(patternString)
)

// Render renders the Compose file for this app, merging in parameters files, other compose files, and env
// appname string, composeFiles []string, parametersFiles []string
func Render(app *types.App, env map[string]string, imageMap map[string]bundle.Image) (*composetypes.Config, error) {
	// prepend the app parameters to the argument parameters
	// load the parameters into a struct
	fileParameters := app.Parameters()
	// inject our metadata
	metaPrefixed, err := parameters.Load(app.MetadataRaw(), parameters.WithPrefix("app"))
	if err != nil {
		return nil, err
	}
	envParameters, err := parameters.FromFlatten(env)
	if err != nil {
		return nil, err
	}
	allParameters, err := parameters.Merge(fileParameters, metaPrefixed, envParameters)
	if err != nil {
		return nil, errors.Wrap(err, "failed to merge parameters")
	}
	composeContent := string(app.Composes()[0])
	composeContent, err = substituteParams(allParameters.Flatten(), composeContent)
	if err != nil {
		return nil, err
	}
	return render(app.Path, composeContent, imageMap)
}

func substituteParams(allParameters map[string]string, composeContent string) (string, error) {
	matches := rePattern.FindAllStringSubmatch(composeContent, -1)
	if len(matches) == 0 {
		return composeContent, nil
	}
	for _, match := range matches {
		groups := make(map[string]string)
		for i, name := range rePattern.SubexpNames()[1:] {
			groups[name] = match[i+1]
		}
		//fail on default values enclosed within {}
		if fail := groups["fail"]; fail != "" {
			return "", errors.New(fmt.Sprintf("Parameters must not have default values set in compose file. Invalid parameter: %s.", match[0]))
		}
		if skip := groups["skip"]; skip != "" {
			continue
		}
		varString := match[0]
		val := groups["named"]
		if val == "" {
			val = groups["braced"]
		}
		if value, ok := allParameters[val]; ok {
			composeContent = strings.ReplaceAll(composeContent, varString, value)
		} else {
			return "", errors.New(fmt.Sprintf("Failed to set value for %s. Value not found in parameters.", val))
		}
	}
	return composeContent, nil
}

func render(appPath string, composeContent string, imageMap map[string]bundle.Image) (*composetypes.Config, error) {
	configFiles, _, err := compose.Load([][]byte{[]byte(composeContent)})
	if err != nil {
		return nil, errors.Wrap(err, "failed to load compose content")
	}
	rendered, err := loader.Load(composetypes.ConfigDetails{
		WorkingDir:  appPath,
		ConfigFiles: configFiles,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to load Compose file")
	}
	if err := processEnabled(rendered); err != nil {
		return nil, err
	}
	for ix, service := range rendered.Services {
		if img, ok := imageMap[service.Name]; ok {
			service.Image = img.Image
			rendered.Services[ix] = service
		}
	}
	return rendered, nil
}

func processEnabled(config *composetypes.Config) error {
	services := []composetypes.ServiceConfig{}
	for _, service := range config.Services {
		if service.Extras != nil {
			if xEnabled, ok := service.Extras["x-enabled"]; ok {
				enabled, err := isEnabled(xEnabled)
				if err != nil {
					return err
				}
				if !enabled {
					continue
				}
			}
		}
		services = append(services, service)
	}
	config.Services = services
	return nil
}

func isEnabled(e interface{}) (bool, error) {
	switch v := e.(type) {
	case string:
		v = strings.ToLower(strings.TrimSpace(v))
		switch {
		case v == "1", v == "true":
			return true, nil
		case v == "", v == "0", v == "false":
			return false, nil
		case strings.HasPrefix(v, "!"):
			nv, err := isEnabled(v[1:])
			if err != nil {
				return false, err
			}
			return !nv, nil
		default:
			return false, errors.Errorf("%s is not a valid value for x-enabled", e)
		}
	case bool:
		return v, nil
	}
	return false, errors.Errorf("invalid type (%T) for x-enabled", e)
}
