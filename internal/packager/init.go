package packager

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/docker/app/internal"
	"github.com/docker/app/internal/compose"
	"github.com/docker/app/internal/validator"
	"github.com/docker/app/internal/yaml"
	"github.com/docker/app/types"
	"github.com/docker/app/types/metadata"
	"github.com/docker/app/types/parameters"
	composeloader "github.com/docker/cli/cli/compose/loader"
	"github.com/docker/cli/cli/compose/schema"
	"github.com/docker/cli/opts"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Init is the entrypoint initialization function.
// It generates a new application definition based on the provided parameters
// and returns the path to the created application definition.
func Init(errWriter io.Writer, name string, composeFile string) (string, error) {
	if err := internal.ValidateAppName(name); err != nil {
		return "", err
	}
	dirName := internal.DirNameFromAppName(name)
	if err := os.Mkdir(dirName, 0755); err != nil {
		return "", errors.Wrap(err, "failed to create application directory")
	}
	var err error
	defer func() {
		if err != nil {
			os.RemoveAll(dirName)
		}
	}()
	if err = writeMetadataFile(name, dirName); err != nil {
		return "", err
	}

	if composeFile == "" {
		err = initFromScratch(name)
	} else {
		v := validator.NewValidatorWithDefaults()
		err = v.Validate(composeFile)
		if err != nil {
			return "", err
		}
		err = initFromComposeFile(errWriter, name, composeFile)
	}
	if err != nil {
		return "", err
	}
	return dirName, nil
}

func initFromScratch(name string) error {
	logrus.Debug("Initializing from scratch")
	composeData, err := composeFileFromScratch()
	if err != nil {
		return err
	}

	dirName := internal.DirNameFromAppName(name)

	if err := ioutil.WriteFile(filepath.Join(dirName, internal.ComposeFileName), composeData, 0644); err != nil {
		return err
	}
	return ioutil.WriteFile(filepath.Join(dirName, internal.ParametersFileName), []byte{'\n'}, 0644)
}

func checkComposeFileVersion(compose map[string]interface{}) error {
	version, ok := compose["version"]
	if !ok {
		return fmt.Errorf("unsupported Compose file version: version 1 is too low")
	}
	return schema.Validate(compose, fmt.Sprintf("%v", version))
}

func getEnvFiles(svcName string, envFileEntry interface{}) ([]string, error) {
	var envFiles []string
	switch envFileEntry := envFileEntry.(type) {
	case string:
		envFiles = append(envFiles, envFileEntry)
	case []interface{}:
		for _, env := range envFileEntry {
			envFiles = append(envFiles, env.(string))
		}
	default:
		return nil, fmt.Errorf("unknown entries in 'env_file' for service %s -> %v",
			svcName, envFileEntry)
	}
	return envFiles, nil
}

func checkEnvFiles(errWriter io.Writer, appName string, cfgMap map[string]interface{}) error {
	services := cfgMap["services"]
	servicesMap, ok := services.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid Compose file")
	}
	for svcName, svc := range servicesMap {
		svcContent, ok := svc.(map[string]interface{})
		if !ok {
			return fmt.Errorf("invalid service %q", svcName)
		}
		envFileEntry, ok := svcContent["env_file"]
		if !ok {
			continue
		}
		envFiles, err := getEnvFiles(svcName, envFileEntry)
		if err != nil {
			return errors.Wrap(err, "invalid Compose file")
		}
		for _, envFilePath := range envFiles {
			fmt.Fprintf(errWriter,
				"%s.env_file %q will not be copied into %s.dockerapp. "+
					"Please copy it manually and update the path accordingly in the compose file.\n",
				svcName, envFilePath, appName)
		}
	}
	return nil
}

func getParamsFromDefaultEnvFile(composeFile string, composeRaw []byte) (map[string]string, bool, error) {
	params := make(map[string]string)
	envs, err := opts.ParseEnvFile(filepath.Join(filepath.Dir(composeFile), ".env"))
	if err == nil {
		for _, v := range envs {
			kv := strings.SplitN(v, "=", 2)
			if len(kv) == 2 {
				params[kv[0]] = kv[1]
			}
		}
	}
	vars, err := compose.ExtractVariables(composeRaw, compose.ExtrapolationPattern)
	if err != nil {
		return nil, false, errors.Wrap(err, "failed to parse compose file")
	}
	needsFilling := false
	for k, v := range vars {
		if _, ok := params[k]; !ok {
			if v != "" {
				params[k] = v
			} else {
				params[k] = "FILL ME"
				needsFilling = true
			}
		}
	}
	return params, needsFilling, nil
}

func initFromComposeFile(errWriter io.Writer, name string, composeFile string) error {
	logrus.Debugf("Initializing from compose file %s", composeFile)

	dirName := internal.DirNameFromAppName(name)

	composeRaw, err := ioutil.ReadFile(composeFile)
	if err != nil {
		return errors.Wrapf(err, "failed to read compose file %q", composeFile)
	}
	cfgMap, err := composeloader.ParseYAML(composeRaw)
	if err != nil {
		return errors.Wrap(err, "failed to parse compose file")
	}
	if err := checkComposeFileVersion(cfgMap); err != nil {
		return err
	}
	if err := checkEnvFiles(errWriter, name, cfgMap); err != nil {
		return err
	}
	params, needsFilling, err := getParamsFromDefaultEnvFile(composeFile, composeRaw)
	if err != nil {
		return err
	}
	expandedParams, err := parameters.FromFlatten(params)
	if err != nil {
		return errors.Wrap(err, "failed to expand parameters")
	}
	parametersYAML, err := yaml.Marshal(expandedParams)
	if err != nil {
		return errors.Wrap(err, "failed to marshal parameters")
	}
	// remove parameter default values from compose before saving
	composeRaw = removeDefaultValuesFromCompose(composeRaw)
	err = ioutil.WriteFile(filepath.Join(dirName, internal.ComposeFileName), composeRaw, 0644)
	if err != nil {
		return errors.Wrap(err, "failed to write docker-compose.yml")
	}
	err = ioutil.WriteFile(filepath.Join(dirName, internal.ParametersFileName), parametersYAML, 0644)
	if err != nil {
		return errors.Wrap(err, "failed to write parameters.yml")
	}
	if needsFilling {
		fmt.Fprintln(os.Stderr, "You will need to edit parameters.yml to fill in default values.")
	}
	return nil
}

func removeDefaultValuesFromCompose(compose []byte) []byte {
	// find variable names followed by default values/error messages with ':-', '-', ':?' and '?' as separators.
	rePattern := regexp.MustCompile(`\$\{[a-zA-Z_]+[a-zA-Z0-9_.]*((:-)|(\-)|(:\?)|(\?))(.*)\}`)
	matches := rePattern.FindAllSubmatch(compose, -1)
	//remove default value from compose content
	for _, groups := range matches {
		variable := groups[0]
		separator := groups[1]
		variableName := bytes.SplitN(variable, separator, 2)[0]
		compose = bytes.ReplaceAll(compose, variable, []byte(fmt.Sprintf("%s}", variableName)))
	}
	return compose
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
# List of application maintainers with name and email for each
{{ if len .Maintainers }}maintainers:
{{ range .Maintainers }}  - name: {{ .Name  }}
    email: {{ .Email }}
{{ end }}{{ else }}#maintainers:
#  - name: John Doe
#    email: john@doe.com
{{ end }}`

func writeMetadataFile(name, dirName string) error {
	meta := newMetadata(name)
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

func newMetadata(name string) metadata.AppMetadata {
	res := metadata.AppMetadata{
		Version: "0.1.0",
		Name:    name,
	}
	userData, _ := user.Current()
	if userData != nil && userData.Username != "" {
		res.Maintainers = []metadata.Maintainer{{Name: userData.Username}}
	}
	return res
}
