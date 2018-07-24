package render

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/docker/app/internal"
	"github.com/docker/app/internal/renderer"
	"github.com/docker/app/internal/settings"
	"github.com/docker/app/internal/slices"
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
)

var (
	delimiter    = "\\$"
	substitution = "[_a-z][._a-z0-9]*(?::?[-?][^}]*)?"

	patternString = fmt.Sprintf(
		"%s(?i:(?P<escaped>%s)|(?P<named>%s)|{(?P<braced>%s)}|(?P<invalid>))",
		delimiter, delimiter, substitution, substitution,
	)

	pattern = regexp.MustCompile(patternString)
)

// Render renders the Compose file for this app, merging in settings files, other compose files, and env
func Render(appname string, composeFiles []string, settingsFiles []string, env map[string]string) (*composetypes.Config, error) {
	// prepend the app settings to the argument settings
	sf := []string{filepath.Join(appname, internal.SettingsFileName)}
	sf = append(sf, settingsFiles...)
	// load the settings into a struct
	fileSettings, err := settings.LoadFiles(sf)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load settings")
	}
	// inject our metadata
	metaFile := filepath.Join(appname, internal.MetadataFileName)
	metaPrefixed, err := settings.LoadFile(metaFile, settings.WithPrefix("app"))
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
	composes := []string{filepath.Join(appname, internal.ComposeFileName)}
	composes = append(composes, composeFiles...)
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
	configFiles := []composetypes.ConfigFile{}
	for _, c := range composes {
		data, err := ioutil.ReadFile(c)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to read Compose file %s", c)
		}
		s, err := renderer.Apply(string(data), allSettings, renderers...)
		if err != nil {
			return nil, err
		}
		parsed, err := loader.ParseYAML([]byte(s))
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse Compose file %s", c)
		}
		configFiles = append(configFiles, composetypes.ConfigFile{Config: parsed})
	}
	return render(configFiles, allSettings.Flatten())
}

func render(configFiles []composetypes.ConfigFile, finalEnv map[string]string) (*composetypes.Config, error) {
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
	processEnabled(rendered)
	return rendered, nil
}

func substitute(template string, mapping composetemplate.Mapping) (string, error) {
	return composetemplate.SubstituteWith(template, mapping, pattern, errorIfMissing)
}

func errorIfMissing(substitution string, mapping composetemplate.Mapping) (string, bool, error) {
	value, found := mapping(substitution)
	if !found {
		return "", true, &composetemplate.InvalidTemplateError{
			Template: "required variable" + substitution + "is missing a value",
		}
	}
	return value, true, nil
}

func processEnabled(config *composetypes.Config) {
	services := []composetypes.ServiceConfig{}
	for _, service := range config.Services {
		if service.Extras != nil {
			if xEnabled, ok := service.Extras["x-enabled"]; ok && !isEnabled(xEnabled.(string)) {
				continue
			}
		}
		services = append(services, service)
	}
	config.Services = services
}

func isEnabled(e string) bool {
	e = strings.ToLower(e)
	switch {
	case e == "", e == "0", e == "false":
		return false
	case strings.HasPrefix(e, "!"):
		return !isEnabled(e[1:])
	default:
		return true
	}
}
