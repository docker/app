package render

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/docker/app/internal/compose"
	"github.com/docker/app/internal/renderer"
	"github.com/docker/app/internal/slices"
	"github.com/docker/app/types"
	"github.com/docker/app/types/settings"
	"github.com/docker/cli/cli/compose/loader"
	composetemplate "github.com/docker/cli/cli/compose/template"
	composetypes "github.com/docker/cli/cli/compose/types"
	"github.com/pkg/errors"

	// Register gotemplate renderer
	_ "github.com/docker/app/internal/renderer/gotemplate"
	// Register mustache renderer
	_ "github.com/docker/app/internal/renderer/mustache"
	// Register yatee renderer
	_ "github.com/docker/app/internal/renderer/yatee"

	// Register json formatter
	_ "github.com/docker/app/internal/formatter/json"
	// Register yaml formatter
	_ "github.com/docker/app/internal/formatter/yaml"
)

var (
	delimiter    = "\\$"
	substitution = "[_a-z][._a-z0-9]*(?::?[-?][^}]*)?"

	patternString = fmt.Sprintf(
		"%s(?i:(?P<escaped>%s)|(?P<named>%s)|{(?P<braced>%s)}|(?P<invalid>))",
		delimiter, delimiter, substitution, substitution,
	)

	// Pattern is the variable regexp pattern used to interpolate or extract variables when rendering
	Pattern = regexp.MustCompile(patternString)
)

// Render renders the Compose file for this app, merging in settings files, other compose files, and env
// appname string, composeFiles []string, settingsFiles []string
func Render(app *types.App, env map[string]string) (*composetypes.Config, error) {
	// prepend the app settings to the argument settings
	// load the settings into a struct
	fileSettings := app.Settings()
	// inject our metadata
	metaPrefixed, err := settings.Load(app.MetadataRaw(), settings.WithPrefix("app"))
	if err != nil {
		return nil, err
	}
	envSettings, err := settings.FromFlatten(env)
	if err != nil {
		return nil, err
	}
	allSettings, err := settings.Merge(fileSettings, metaPrefixed, envSettings)
	if err != nil {
		return nil, errors.Wrap(err, "failed to merge settings")
	}
	// prepend our app compose file to the list
	renderers := renderer.Drivers()
	if r, ok := os.LookupEnv("DOCKERAPP_RENDERERS"); ok {
		rl := strings.Split(r, ",")
		for _, r := range rl {
			if !slices.ContainsString(renderer.Drivers(), r) {
				return nil, fmt.Errorf("renderer '%s' not found", r)
			}
		}
		renderers = rl
	}
	configFiles, err := compose.Load(app.Composes(), func(data string) (string, error) {
		return renderer.Apply(data, allSettings, renderers...)
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to load composefiles")
	}
	return render(configFiles, allSettings.Flatten(), app.DisabledServices)
}

func render(configFiles []composetypes.ConfigFile, finalEnv map[string]string, disabled map[string]bool) (*composetypes.Config, error) {
	rendered, err := loader.Load(composetypes.ConfigDetails{
		WorkingDir:  ".",
		ConfigFiles: configFiles,
		Environment: finalEnv,
	}, func(opts *loader.Options) {
		opts.Interpolate.Substitute = substitute
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to load Compose file")
	}
	if err := processEnabled(rendered, disabled); err != nil {
		return nil, err
	}
	return rendered, nil
}

func substitute(template string, mapping composetemplate.Mapping) (string, error) {
	return composetemplate.SubstituteWith(template, mapping, Pattern, errorIfMissing)
}

func errorIfMissing(substitution string, mapping composetemplate.Mapping) (string, bool, error) {
	value, found := mapping(substitution)
	if !found {
		return "", true, &composetemplate.InvalidTemplateError{
			Template: "required variable " + substitution + " is missing a value",
		}
	}
	return value, true, nil
}

func processEnabled(config *composetypes.Config, disabled map[string]bool) error {
	services := []composetypes.ServiceConfig{}
	for _, service := range config.Services {
		if disabled != nil && disabled[service.Name] {
			continue
		}

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
