package renderer

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"

	conversion "github.com/docker/cli/cli/command/stack/kubernetes"
	"github.com/docker/cli/kubernetes/compose/v1beta2"
	"github.com/docker/lunchbox/packager"
	"github.com/docker/lunchbox/types"
	"github.com/docker/lunchbox/utils"
	yaml "gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type helmMaintainer struct {
	Name string
}

type helmMeta struct {
	Name        string
	Version     string
	Description string
	Keywords    []string
	Maintainers []helmMaintainer
}

func toHelmMeta(meta *types.AppMetadata) (*helmMeta, error) {
	return &helmMeta{
		Name:        meta.Name,
		Version:     meta.Version,
		Description: meta.Description,
		Maintainers: []helmMaintainer{{Name: meta.Author}},
	}, nil
}

func mergeValues(target map[interface{}]interface{}, source map[string]interface{}) {
	for k, v := range source {
		tv, ok := target[k]
		if !ok {
			target[k] = v
			continue
		}
		switch tvv := tv.(type) {
		case map[interface{}]interface{}:
			mergeValues(tvv, v.(map[string]interface{}))
		default:
			target[k] = v
		}
	}
}

func contains(list []string, needle string) bool {
	for _, e := range list {
		if e == needle {
			return true
		}
	}
	return false
}

// remove from settings all stuff that is not in variables
func filterVariables(settings map[string]interface{}, variables []string, prefix string) {
	for k, v := range settings {
		switch vv := v.(type) {
		case map[string]interface{}:
			filterVariables(vv, variables, prefix+k+".")
			if len(vv) == 0 {
				delete(settings, k)
			}
		default:
			if !contains(variables, prefix+k) {
				delete(settings, k)
			}
		}
	}
}

// Helm renders an app as an Helm Chart
func Helm(appname string, composeFiles []string, settingsFile []string, env map[string]string, templated bool) error {
	oAppname := appname
	appname, cleanup, err := packager.Extract(appname)
	if err != nil {
		return err
	}
	defer cleanup()
	// Render once to get the list of variables
	rendered, flatSettings, settings, err := RenderEx(appname, composeFiles, settingsFile, env, false)
	if err != nil {
		return err
	}
	var listSettings []string
	for k := range flatSettings {
		listSettings = append(listSettings, k)
	}
	// create placeholders for all variables
	placeholders := make(map[string]string)
	placeHolderBase := 44100
	for i, v := range listSettings {
		placeholders[v] = fmt.Sprintf("%v", placeHolderBase+i)
	}
	// render with placeholders
	renderedPH, err := Render(appname, composeFiles, nil, placeholders)
	if err != nil {
		return err
	}
	// read our metadata
	metaFile := path.Join(appname, "metadata.yml")
	metaContent, err := ioutil.ReadFile(metaFile)
	if err != nil {
		return err
	}
	var meta types.AppMetadata
	err = yaml.Unmarshal(metaContent, &meta)
	if err != nil {
		return err
	}
	targetDir := utils.AppNameFromDir(oAppname) + ".chart"
	os.Mkdir(targetDir, 0755)
	// Chart.yaml: extract from app metadata and merge with existing content
	hmeta, err := toHelmMeta(&meta)
	if err != nil {
		return err
	}
	chart := make(map[interface{}]interface{})
	prevChartRaw, err := ioutil.ReadFile(path.Join(targetDir, "Chart.yaml"))
	if err == nil {
		err = yaml.Unmarshal(prevChartRaw, chart)
		if err != nil {
			return err
		}
	}
	chart["name"] = hmeta.Name
	chart["version"] = hmeta.Version
	chart["description"] = hmeta.Description
	chart["keywords"] = hmeta.Keywords
	chart["maintainers"] = hmeta.Maintainers
	hmetadata, err := yaml.Marshal(chart)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(path.Join(targetDir, "Chart.yaml"), hmetadata, 0644)
	if err != nil {
		return err
	}
	// Write stack in templates/
	os.Mkdir(path.Join(targetDir, "templates"), 0755)
	if !templated {
		renderedPH = rendered
	}
	stackSpec := conversion.FromComposeConfig(renderedPH)
	stack := v1beta2.Stack{
		TypeMeta: metav1.TypeMeta{
			Kind:       "stacks.compose.docker.com",
			APIVersion: "v1beta2",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      utils.AppNameFromDir(appname),
			Namespace: "default", // FIXME
		},
		Spec: stackSpec,
	}
	stackData, err := yaml.Marshal(stack)
	if err != nil {
		return err
	}
	if templated {
		// Replace all placeholder values with the matching variable
		re := regexp.MustCompile(`(44[0-9][0-9][0-9])`)
		hits := re.FindAll(stackData, -1)
		for _, h := range hits {
			hs := string(h)
			idx, _ := strconv.Atoi(hs)
			stackData = []byte(strings.Replace(string(stackData), hs,
				"{{."+listSettings[idx-placeHolderBase]+"}}", -1))
		}
	}
	err = ioutil.WriteFile(path.Join(targetDir, "templates", "stack.yaml"), stackData, 0644)
	if err != nil {
		return err
	}
	// filter settings with variables actually used in compose files
	cf, err := ioutil.ReadFile(path.Join(appname, "docker-compose.yml"))
	if err != nil {
		return err
	}
	variables, err := packager.ExtractVariables(string(cf))
	if err != nil {
		return err
	}
	filterVariables(settings, variables, "")
	// merge settings with existing values.yml
	values := make(map[interface{}]interface{})
	if valuesCur, err := ioutil.ReadFile(path.Join(targetDir, "values.yaml")); err == nil {
		err = yaml.Unmarshal(valuesCur, values)
		if err != nil {
			return err
		}
	}
	mergeValues(values, settings)
	valuesRaw, err := yaml.Marshal(values)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(path.Join(targetDir, "values.yaml"), valuesRaw, 0644)
	return err
}
