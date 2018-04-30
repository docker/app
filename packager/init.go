package packager

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/docker/lunchbox/types"
	"github.com/docker/lunchbox/utils"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

// Init is the entrypoint initialization function.
// It generates a new application package based on the provided parameters.
func Init(name string, composeFile string) error {
	if err := utils.ValidateAppName(name); err != nil {
		return err
	}
	dirName := utils.DirNameFromAppName(name)
	if err := os.Mkdir(dirName, 0755); err != nil {
		return err
	}
	var err error
	defer func() {
		if err != nil {
			os.RemoveAll(dirName)
		}
	}()
	if err = writeMetadataFile(name, dirName); err != nil {
		return err
	}

	if composeFile == "" {
		if _, err := os.Stat("docker-compose.yml"); err == nil {
			composeFile = "docker-compose.yml"
		}
	}
	if composeFile == "" {
		err = initFromScratch(name)
	} else {
		err = initFromComposeFile(name, composeFile)
	}
	return err
}

func initFromScratch(name string) error {
	log.Debug("init from scratch")
	composeData, err := composeFileFromScratch()
	if err != nil {
		return err
	}

	dirName := utils.DirNameFromAppName(name)
	if err := utils.CreateFileWithData(filepath.Join(dirName, "docker-compose.yml"), composeData); err != nil {
		return err
	}
	return utils.CreateFileWithData(filepath.Join(dirName, "settings.yml"), []byte{'\n'})
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
	return (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z') || (b >= '0' && b <= '9') || b == '_'
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

func extractVariables(composeRaw string) ([]string, error) {
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

	dirName := utils.DirNameFromAppName(name)
	composeRaw, err := ioutil.ReadFile(composeFile)
	if err != nil {
		return errors.Wrap(err, "failed to read compose file")
	}
	settings := make(map[string]string)
	envRaw, err := ioutil.ReadFile(filepath.Join(filepath.Dir(composeFile), ".env"))
	if err == nil {
		parseEnv(string(envRaw), settings)
	}
	keys, err := extractVariables(string(composeRaw))
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
	err = ioutil.WriteFile(filepath.Join(dirName, "docker-compose.yml"), composeRaw, 0644)
	if err != nil {
		return errors.Wrap(err, "failed to write docker-compose.yml")
	}
	err = ioutil.WriteFile(filepath.Join(dirName, "settings.yml"), settingsYAML, 0644)
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

func writeMetadataFile(name, dirName string) error {
	data, err := yaml.Marshal(newMetadata(name))
	if err != nil {
		return err
	}
	return utils.CreateFileWithData(filepath.Join(dirName, "metadata.yml"), data)
}

func newMetadata(name string) types.AppMetadata {
	var userName string
	target := types.ApplicationTarget{
		Swarm:      true,
		Kubernetes: true,
	}
	userData, _ := user.Current()
	if userData != nil {
		userName = userData.Username
	}
	return types.AppMetadata{
		Version: "0.1.0",
		Targets: target,
		Name:    name,
		Author:  userName,
	}
}
