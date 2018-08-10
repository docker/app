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
	"github.com/docker/app/internal/compose"
	"github.com/docker/app/loader"
	"github.com/docker/app/render"
	"github.com/docker/app/types"
	"github.com/docker/app/types/metadata"
	composeloader "github.com/docker/cli/cli/compose/loader"
	"github.com/docker/cli/cli/compose/schema"
	"github.com/docker/cli/opts"
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
	app, err := loader.LoadFromDirectory(temp)
	if err != nil {
		return err
	}
	return Merge(app, target)
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

func checkComposeFileVersion(compose map[string]interface{}) error {
	version, ok := compose["version"]
	if !ok {
		return fmt.Errorf("unsupported Compose file version: version 1 is too low")
	}
	return schema.Validate(compose, fmt.Sprintf("%v", version))
}

func initFromComposeFile(name string, composeFile string) error {
	log.Debug("init from compose")

	dirName := internal.DirNameFromAppName(name)

	composeRaw, err := ioutil.ReadFile(composeFile)
	if err != nil {
		return errors.Wrap(err, "failed to read compose file")
	}
	cfgMap, err := composeloader.ParseYAML(composeRaw)
	if err != nil {
		return errors.Wrap(err, "failed to parse compose file")
	}
	if err := checkComposeFileVersion(cfgMap); err != nil {
		return err
	}
	settings := make(map[string]string)
	envs, err := opts.ParseEnvFile(filepath.Join(filepath.Dir(composeFile), ".env"))
	if err == nil {
		for _, v := range envs {
			kv := strings.SplitN(v, "=", 2)
			if len(kv) == 2 {
				settings[kv[0]] = kv[1]
			}
		}
	}
	vars, err := compose.ExtractVariables(composeRaw, render.Pattern)
	if err != nil {
		return errors.Wrap(err, "failed to parse compose file")
	}
	needsFilling := false
	for k, v := range vars {
		if _, ok := settings[k]; !ok {
			if v != "" {
				settings[k] = v
			} else {
				settings[k] = "FILL ME"
				needsFilling = true
			}
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
# List of application maintainers with name and email for each
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

// parseMaintainersData parses user-provided data through the maintainers flag and returns
// a slice of Maintainer instances
func parseMaintainersData(maintainers []string) []metadata.Maintainer {
	var res []metadata.Maintainer
	for _, m := range maintainers {
		ne := strings.SplitN(m, ":", 2)
		var email string
		if len(ne) > 1 {
			email = ne[1]
		}
		res = append(res, metadata.Maintainer{Name: ne[0], Email: email})
	}
	return res
}

func newMetadata(name string, description string, maintainers []string) metadata.AppMetadata {
	res := metadata.AppMetadata{
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
		res.Maintainers = []metadata.Maintainer{{Name: userName}}
	} else {
		res.Maintainers = parseMaintainersData(maintainers)
	}
	return res
}
