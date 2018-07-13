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
	"github.com/docker/cli/cli/compose/loader"
	composetemplate "github.com/docker/cli/cli/compose/template"
	composetypes "github.com/docker/cli/cli/compose/types"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"

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

//flattenYAML reads a YAML file and return a flattened view
func flattenYAML(content []byte) (map[string]string, error) {
	in := make(map[interface{}]interface{})
	if err := yaml.Unmarshal(content, in); err != nil {
		return nil, err
	}
	out := make(map[string]interface{})
	merge(out, in)
	res := make(map[string]string)
	flatten(out, res, "")
	return res, nil
}

// flatten flattens a structure: foo.bar.baz -> foo_bar_baz
func flatten(in map[string]interface{}, out map[string]string, prefix string) {
	for k, v := range in {
		switch vv := v.(type) {
		case string:
			out[prefix+k] = vv
		case map[string]interface{}:
			flatten(vv, out, prefix+k+".")
		default:
			out[prefix+k] = fmt.Sprintf("%v", v)
		}
	}
}

func merge(res map[string]interface{}, src map[interface{}]interface{}) {
	for k, v := range src {
		kk, ok := k.(string)
		if !ok {
			panic(fmt.Sprintf("fatal error, key %v in %#v is not a string", k, src))
		}
		eval, ok := res[kk]
		switch vv := v.(type) {
		case map[interface{}]interface{}:
			if !ok {
				res[kk] = make(map[string]interface{})
			} else {
				if _, ok2 := eval.(map[string]interface{}); !ok2 {
					res[kk] = make(map[string]interface{})
				}
			}
			merge(res[kk].(map[string]interface{}), vv)
		default:
			res[kk] = vv
		}
	}
}

// LoadSettings loads a set of settings file and produce a property dictionary
func loadSettings(files []string) (map[string]interface{}, error) {
	res := make(map[string]interface{})
	for _, f := range files {
		data, err := ioutil.ReadFile(f)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to read settings file %s", f)
		}
		s := make(map[interface{}]interface{})
		err = yaml.Unmarshal(data, &s)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse settings file %s", f)
		}
		merge(res, s)
	}
	return res, nil
}

// MergeSettings merges a flattened settings map into an expanded one
func mergeSettings(settings map[string]interface{}, env map[string]string) error {
	for k, v := range env {
		ss := strings.Split(k, ".")
		valroot := make(map[interface{}]interface{})
		val := valroot
		for _, s := range ss[:len(ss)-1] {
			m := make(map[interface{}]interface{})
			val[s] = m
			val = m
		}
		var converted interface{}
		err := yaml.Unmarshal([]byte(v), &converted)
		if err != nil {
			return err
		}
		val[ss[len(ss)-1]] = converted
		merge(settings, valroot)
	}
	return nil
}

func contains(list []string, needle string) bool {
	for _, e := range list {
		if e == needle {
			return true
		}
	}
	return false
}

// Render renders the Compose file for this app, merging in settings files, other compose files, and env
func Render(appname string, composeFiles []string, settingsFiles []string, env map[string]string) (*composetypes.Config, error) {
	// prepend the app settings to the argument settings
	sf := []string{filepath.Join(appname, internal.SettingsFileName)}
	sf = append(sf, settingsFiles...)
	// load the settings into a struct
	settings, err := loadSettings(sf)
	if err != nil {
		return nil, err
	}
	// inject our metadata
	metaFile := filepath.Join(appname, internal.MetadataFileName)
	meta := make(map[interface{}]interface{})
	metaContent, err := ioutil.ReadFile(metaFile)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read package metadata file")
	}
	err = yaml.Unmarshal(metaContent, &meta)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse package metadata file")
	}
	metaPrefixed := make(map[interface{}]interface{})
	metaPrefixed["app"] = meta
	merge(settings, metaPrefixed)
	// inject the user-provided env
	err = mergeSettings(settings, env)
	if err != nil {
		return nil, errors.Wrap(err, "failed to merge settings")
	}
	// flatten settings for variable expension
	finalEnv := make(map[string]string)
	flatten(settings, finalEnv, "")
	// prepend our app compose file to the list
	composes := []string{filepath.Join(appname, internal.ComposeFileName)}
	composes = append(composes, composeFiles...)
	renderers := renderer.Drivers()
	if r, ok := os.LookupEnv("DOCKERAPP_RENDERERS"); ok {
		rl := strings.Split(r, ",")
		for _, r := range rl {
			if !contains(renderer.Drivers(), r) {
				return nil, fmt.Errorf("renderer '%s' not found", r)
			}
		}
		renderers = rl
	}
	// go-template, then parse, then expand the compose files
	configFiles := []composetypes.ConfigFile{}
	for _, c := range composes {
		data, err := ioutil.ReadFile(c)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to read Compose file %s", c)
		}
		s, err := renderer.Apply(string(data), settings, renderers...)
		if err != nil {
			return nil, err
		}
		parsed, err := loader.ParseYAML([]byte(s))
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse Compose file %s", c)
		}
		configFiles = append(configFiles, composetypes.ConfigFile{Config: parsed})
	}
	return render(configFiles, finalEnv)
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
