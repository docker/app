package packager

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/docker/app/internal"
	"github.com/docker/app/internal/types"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

func prependToFile(filename, text string) error {
	content, _ := ioutil.ReadFile(filename)
	content = append([]byte(text), content...)
	return ioutil.WriteFile(filename, content, 0644)
}

// Init is the entrypoint initialization function.
// It generates a new application package based on the provided parameters.
func Init(name string, composeFile string, description string, maintainers []string, singleFile bool) error {
	if err := internal.ValidateAppName(name); err != nil {
		return err
	}
	dirName := internal.DirNameFromAppName(name)
	if err := os.Mkdir(dirName, 0755); err != nil {
		return errors.Wrap(err, "failed to create application directory")
	}
	var err error
	defer func() {
		if err != nil {
			os.RemoveAll(dirName)
		}
	}()
	if err = writeMetadataFile(name, dirName, description, maintainers); err != nil {
		return err
	}

	if composeFile == "" {
		if _, err := os.Stat(internal.ComposeFileName); err == nil {
			composeFile = internal.ComposeFileName
		}
	}
	if composeFile == "" {
		err = initFromScratch(name)
	} else {
		err = initFromComposeFile(name, composeFile)
	}
	if err != nil {
		return err
	}
	if !singleFile {
		return nil
	}
	// Merge as a single file
	// Add some helfpful comments to distinguish the sections
	if err := prependToFile(filepath.Join(dirName, internal.ComposeFileName), "# This section contains the Compose file that describes your application services.\n"); err != nil {
		return err
	}
	if err := prependToFile(filepath.Join(dirName, internal.SettingsFileName), "# This section contains the default values for your application settings.\n"); err != nil {
		return err
	}
	if err := prependToFile(filepath.Join(dirName, internal.MetadataFileName), "# This section contains your application metadata.\n"); err != nil {
		return err
	}

	temp := "_temp_dockerapp__.dockerapp"
	err = os.Rename(dirName, temp)
	if err != nil {
		return err
	}
	defer os.RemoveAll(temp)
	var target io.Writer
	target, err = os.Create(dirName)
	if err != nil {
		return err
	}
	defer target.(io.WriteCloser).Close()
	return Merge(temp, target)
}

func initFromScratch(name string) error {
	log.Debug("init from scratch")
	composeData, err := composeFileFromScratch()
	if err != nil {
		return err
	}

	dirName := internal.DirNameFromAppName(name)

	if err := ioutil.WriteFile(filepath.Join(dirName, internal.ComposeFileName), composeData, 0644); err != nil {
		return err
	}
	return ioutil.WriteFile(filepath.Join(dirName, internal.SettingsFileName), []byte{'\n'}, 0644)
}

func parseEnv(env string, target map[string]string) {
	envlines := strings.Split(env, "\n")
	for _, l := range envlines {
		l = strings.Trim(l, "\r ")
		if l == "" || l[0] == '#' {
			continue
		}
		kv := strings.SplitN(l, "=", 2)
		if len(kv) != 2 {
			continue
		}
		target[kv[0]] = kv[1]
	}
}

func isAlNum(b byte) bool {
	return (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z') || (b >= '0' && b <= '9') || b == '_' || b == '.'
}

func extractString(data string, res *[]string) {
	for {
		dollar := strings.Index(data, "$")
		if dollar == -1 || len(data) == dollar+1 {
			break
		}
		if data[dollar+1] == '$' {
			data = data[dollar+2:]
			continue
		}
		dollar++
		if data[dollar] == '{' {
			dollar++
		}
		start := dollar
		for dollar < len(data) && isAlNum(data[dollar]) {
			dollar++
		}
		*res = append(*res, data[start:dollar])
		data = data[dollar:]
	}
}

func extractRecurseList(node []interface{}, res *[]string) error {
	for _, v := range node {
		switch vv := v.(type) {
		case string:
			extractString(vv, res)
		case []interface{}:
			if err := extractRecurseList(vv, res); err != nil {
				return err
			}
		case map[interface{}]interface{}:
			if err := extractRecurse(vv, res); err != nil {
				return err
			}
		}
	}
	return nil
}

func extractRecurse(node map[interface{}]interface{}, res *[]string) error {
	for _, v := range node {
		switch vv := v.(type) {
		case string:
			extractString(vv, res)
		case []interface{}:
			if err := extractRecurseList(vv, res); err != nil {
				return err
			}
		case map[interface{}]interface{}:
			if err := extractRecurse(vv, res); err != nil {
				return err
			}
		}
	}
	return nil
}

// ExtractVariables returns the list of variables used by given compose raw data
func ExtractVariables(composeRaw string) ([]string, error) {
	compose := make(map[interface{}]interface{})
	err := yaml.Unmarshal([]byte(composeRaw), compose)
	if err != nil {
		return nil, err
	}
	var res []string
	err = extractRecurse(compose, &res)
	return res, err
}

func initFromComposeFile(name string, composeFile string) error {
	log.Debug("init from compose")

	dirName := internal.DirNameFromAppName(name)
	composeRaw, err := ioutil.ReadFile(composeFile)
	if err != nil {
		return errors.Wrap(err, "failed to read compose file")
	}
	settings := make(map[string]string)
	envRaw, err := ioutil.ReadFile(filepath.Join(filepath.Dir(composeFile), ".env"))
	if err == nil {
		parseEnv(string(envRaw), settings)
	}
	keys, err := ExtractVariables(string(composeRaw))
	if err != nil {
		return errors.Wrap(err, "failed to parse compose file")
	}
	needsFilling := false
	for _, k := range keys {
		if _, ok := settings[k]; !ok {
			settings[k] = "FILL ME"
			needsFilling = true
		}
	}
	settingsYAML, err := yaml.Marshal(settings)
	if err != nil {
		return errors.Wrap(err, "failed to marshal settings")
	}
	err = ioutil.WriteFile(filepath.Join(dirName, internal.ComposeFileName), composeRaw, 0644)
	if err != nil {
		return errors.Wrap(err, "failed to write docker-compose.yml")
	}
	err = ioutil.WriteFile(filepath.Join(dirName, internal.SettingsFileName), settingsYAML, 0644)
	if err != nil {
		return errors.Wrap(err, "failed to write settings.yml")
	}
	if needsFilling {
		fmt.Println("You will need to edit settings.yml to fill in default values.")
	}
	return nil
}

func composeFileFromScratch() ([]byte, error) {
	fileStruct := types.NewInitialComposeFile()
	return yaml.Marshal(fileStruct)
}

const metaTemplate = `# Version of the application
version: {{ .Version }}
# Name of the application
name: {{ .Name }}
# A short description of the application
description: {{ .Description }}
# Namespace to use when pushing to a registry. This is typically your Hub username.
{{ if len .Namespace}}namespace: {{ .Namespace }} {{ else }}#namespace: myHubUsername{{ end }}
# List of application maitainers with name and email for each
{{ if len .Maintainers }}maintainers:
{{ range .Maintainers }}  - name: {{ .Name  }}
    email: {{ .Email }}
{{ end }}{{ else }}#maintainers:
#  - name: John Doe
#    email: john@doe.com
{{ end }}`

func writeMetadataFile(name, dirName string, description string, maintainers []string) error {
	meta := newMetadata(name, description, maintainers)
	tmpl, err := template.New("metadata").Parse(metaTemplate)
	if err != nil {
		return errors.Wrap(err, "internal error parsing metadata template")
	}
	resBuf := &bytes.Buffer{}
	if err := tmpl.Execute(resBuf, meta); err != nil {
		return errors.Wrap(err, "error generating metadata")
	}
	return ioutil.WriteFile(filepath.Join(dirName, internal.MetadataFileName), resBuf.Bytes(), 0644)
}

func newMetadata(name string, description string, maintainers []string) types.AppMetadata {
	res := types.AppMetadata{
		Version:     "0.1.0",
		Name:        name,
		Description: description,
	}
	if len(maintainers) == 0 {
		var userName string
		userData, _ := user.Current()
		if userData != nil {
			userName = userData.Username
		}
		res.Maintainers = []types.Maintainer{{Name: userName}}
	} else {
		for _, m := range maintainers {
			ne := strings.Split(m, ":")
			email := ""
			if len(ne) > 1 {
				email = ne[1]
			}
			res.Maintainers = append(res.Maintainers, types.Maintainer{Name: ne[0], Email: email})
		}
	}
	return res
}
