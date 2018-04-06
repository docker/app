package packager

import (
    "fmt"
    "io/ioutil"
    "path"

    "github.com/docker/cli/cli/compose/loader"
    composetypes "github.com/docker/cli/cli/compose/types"
    "gopkg.in/yaml.v2"
)

// inject flattens a structure: foo.bar.baz -> foo_bar_baz
func inject(in map[interface{}]interface{}, out map[string]string, prefix string) {
	for k, v := range in {
		kk := k.(string)
		switch vv := v.(type) {
		case string:
			out[prefix + kk] = vv
		case map[interface{}]interface{}:
			inject(vv, out, prefix + kk + ".")
		default:
			out[prefix + kk] = fmt.Sprintf("%v", v)
		}
	}
}

// load a set of settings file and produce a property dictionary
func loadSettings(files []string) (map[string]string, error) {
	res := make(map[string]string)
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
		inject(s, res, "")
	}
	return res, nil
}

// Render renders the composefile for this app, merging in settings files, other compose files, end env
func Render(appname string, composeFiles []string, settingsFile []string, env map[string]string) (string, error) {
	appname, cleanup, err := Extract(appname)
	if err != nil {
		return "", err
	}
	defer cleanup()
	// prepend the app settings to the argument settings
	sf := []string { path.Join(appname, "settings.yml")}
	sf = append(sf, settingsFile...)
	finalEnv, err := loadSettings(sf)
	for k, v := range(env) {
		finalEnv[k] = v
	}
	mainCompose, err := ioutil.ReadFile(path.Join(appname, "services.yml"))
	if err != nil {
		return "", err
	}
	mainParsed, err := loader.ParseYAML(mainCompose)
	if err != nil {
		return "", err
	}
	configFiles := []composetypes.ConfigFile{ {Config: mainParsed}}
	for _, c := range composeFiles {
		data, err := ioutil.ReadFile(c)
		if err != nil {
			return "", err
		}
		parsed, err := loader.ParseYAML(data)
		if err != nil {
			return "", err
		}
		configFiles = append(configFiles, composetypes.ConfigFile{Config: parsed})
	}
	fmt.Printf("ENV: %v\n", finalEnv)
	rendered, err := loader.Load(composetypes.ConfigDetails {
			WorkingDir: ".",
			ConfigFiles: configFiles,
			Environment: finalEnv,
	})
	if err != nil {
		return "", err
	}
	res, err := yaml.Marshal(rendered)
	return string(res), err
}