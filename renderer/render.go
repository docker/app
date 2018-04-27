package renderer

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"path"
	"strings"
	"text/template"

	"github.com/docker/cli/cli/compose/loader"
	composetypes "github.com/docker/cli/cli/compose/types"
	"github.com/docker/lunchbox/packager"
	"gopkg.in/yaml.v2"
)

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
			panic(fmt.Sprintf("DAFUCK, key %v in %#v is not a string", k, src))
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

// load a set of settings file and produce a property dictionary
func loadSettings(files []string) (map[string]interface{}, error) {
	res := make(map[string]interface{})
	for _, f := range files {
		data, err := ioutil.ReadFile(f)
		if err != nil {
			return nil, err
		}
		s := make(map[interface{}]interface{})
		err = yaml.Unmarshal(data, &s)
		if err != nil {
			return nil, err
		}
		merge(res, s)
	}
	return res, nil
}

// Render renders the Compose file for this app, merging in settings files, other compose files, end env
func Render(appname string, composeFiles []string, settingsFile []string, env map[string]string) (*composetypes.Config, error) {
	comp, _, _, err := RenderEx(appname, composeFiles, settingsFile, env, false)
	return comp, err
}

// RenderEx is like Render, but returns the final env and can escape variables
func RenderEx(appname string, composeFiles []string, settingsFile []string, env map[string]string, escape bool) (*composetypes.Config, map[string]string, map[string]interface{}, error) {
	appname, cleanup, err := packager.Extract(appname)
	if err != nil {
		return nil, nil, nil, err
	}
	defer cleanup()
	// prepend the app settings to the argument settings
	sf := []string{path.Join(appname, "settings.yml")}
	sf = append(sf, settingsFile...)
	// load the settings into a struct
	settings, err := loadSettings(sf)
	if err != nil {
		return nil, nil, nil, err
	}
	// inject our metadata
	metaFile := path.Join(appname, "metadata.yml")
	meta := make(map[interface{}]interface{})
	metaContent, err := ioutil.ReadFile(metaFile)
	if err != nil {
		return nil, nil, nil, err
	}
	err = yaml.Unmarshal(metaContent, &meta)
	if err != nil {
		return nil, nil, nil, err
	}
	metaPrefixed := make(map[interface{}]interface{})
	metaPrefixed["app"] = meta
	merge(settings, metaPrefixed)
	// inject the user-provided env
	for k, v := range env {
		ss := strings.Split(k, ".")
		valroot := make(map[interface{}]interface{})
		val := valroot
		for _, s := range ss[:len(ss)-1] {
			val[s] = make(map[interface{}]interface{})
			val = val[s].(map[interface{}]interface{})
		}
		var converted interface{}
		err = yaml.Unmarshal([]byte(v), &converted)
		if err != nil {
			return nil, nil, nil, err
		}
		val[ss[len(ss)-1]] = converted
		merge(settings, valroot)
	}
	// flatten settings for variable expension
	finalEnv := make(map[string]string)
	flatten(settings, finalEnv, "")
	// prepend our app compose file to the list
	composes := []string{path.Join(appname, "docker-compose.yml")}
	composes = append(composes, composeFiles...)
	// go-template, then parse, then expand the compose files
	configFiles := []composetypes.ConfigFile{}
	for _, c := range composes {
		data, err := ioutil.ReadFile(c)
		if err != nil {
			return nil, nil, nil, err
		}
		if escape {
			data = []byte(strings.Replace(string(data), "$", "$$", -1))
		}
		tmpl, err := template.New("compose").Parse(string(data))
		if err != nil {
			return nil, nil, nil, err
		}
		yaml := bytes.NewBuffer(nil)
		err = tmpl.Execute(yaml, settings)
		if err != nil {
			return nil, nil, nil, err
		}
		parsed, err := loader.ParseYAML(yaml.Bytes())
		if err != nil {
			return nil, nil, nil, err
		}
		configFiles = append(configFiles, composetypes.ConfigFile{Config: parsed})
	}
	//fmt.Printf("ENV: %v\n", finalEnv)
	//fmt.Printf("MAPENV: %#v\n", settings)
	rendered, err := loader.Load(composetypes.ConfigDetails{
		WorkingDir:  ".",
		ConfigFiles: configFiles,
		Environment: finalEnv,
	})
	return rendered, finalEnv, settings, err
}
