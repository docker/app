package renderer

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/cbroglie/mustache"
	"github.com/docker/app/internal"
	"github.com/docker/app/internal/packager"
	"github.com/docker/app/internal/yatee"
	"github.com/docker/cli/cli/compose/loader"
	composetypes "github.com/docker/cli/cli/compose/types"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
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
func LoadSettings(files []string) (map[string]interface{}, error) {
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
func MergeSettings(settings map[string]interface{}, env map[string]string) error {
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

func applyRenderers(data []byte, renderers []string, settings map[string]interface{}) ([]byte, error) {
	for _, r := range renderers {
		switch r {
		case "gotemplate":
			tmpl, err := template.New("compose").Parse(string(data))
			if err != nil {
				return nil, err
			}
			tmpl.Option("missingkey=error")
			yaml := bytes.NewBuffer(nil)
			err = tmpl.Execute(yaml, settings)
			if err != nil {
				return nil, errors.Wrap(err, "failed to execute gotemplate")
			}
			data = yaml.Bytes()
		case "yatee":
			yateed, err := yatee.Process(string(data), settings, yatee.OptionErrOnMissingKey)
			if err != nil {
				return nil, err
			}
			m, err := yaml.Marshal(yateed)
			if err != nil {
				return nil, errors.Wrap(err, "failed to execute yatee template")
			}
			data = []byte(strings.Replace(string(m), "$", "$$", -1))
		case "mustache":
			mdata, err := mustache.Render(string(data), settings)
			if err != nil {
				return nil, errors.Wrap(err, "failed to execute mustache template")
			}
			data = []byte(mdata)
		case "none":
		default:
			return nil, fmt.Errorf("unknown renderer %s", r)
		}
	}
	return data, nil
}

func contains(list []string, needle string) bool {
	for _, e := range list {
		if e == needle {
			return true
		}
	}
	return false
}

// Render renders the Compose file for this app, merging in settings files, other compose files, end env
func Render(appname string, composeFiles []string, settingsFile []string, env map[string]string) (*composetypes.Config, error) {
	appname, cleanup, err := packager.Extract(appname)
	if err != nil {
		return nil, err
	}
	defer cleanup()
	// prepend the app settings to the argument settings
	sf := []string{filepath.Join(appname, "settings.yml")}
	sf = append(sf, settingsFile...)
	// load the settings into a struct
	settings, err := LoadSettings(sf)
	if err != nil {
		return nil, err
	}
	// inject our metadata
	metaFile := filepath.Join(appname, "metadata.yml")
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
	err = MergeSettings(settings, env)
	if err != nil {
		return nil, errors.Wrap(err, "failed to merge settings")
	}
	// flatten settings for variable expension
	finalEnv := make(map[string]string)
	flatten(settings, finalEnv, "")
	// prepend our app compose file to the list
	composes := []string{filepath.Join(appname, "docker-compose.yml")}
	composes = append(composes, composeFiles...)
	renderers := strings.Split(internal.Renderers, ",")
	if r, ok := os.LookupEnv("DOCKERAPP_RENDERERS"); ok {
		rl := strings.Split(r, ",")
		for _, r := range rl {
			if r != "none" && !contains(renderers, r) {
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
		data, err = applyRenderers(data, renderers, settings)
		if err != nil {
			return nil, err
		}
		parsed, err := loader.ParseYAML(data)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse Compose file %s", c)
		}
		configFiles = append(configFiles, composetypes.ConfigFile{Config: parsed})
	}
	//fmt.Printf("ENV: %v\n", finalEnv)
	//fmt.Printf("MAPENV: %#v\n", settings)
	rendered, err := loader.Load(composetypes.ConfigDetails{
		WorkingDir:           ".",
		ConfigFiles:          configFiles,
		Environment:          finalEnv,
		ErrOnMissingVariable: true,
	})
	return rendered, errors.Wrap(err, "failed to load Compose file")
}
