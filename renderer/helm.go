package renderer

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/docker/lunchbox/templateconversion"
	"github.com/docker/lunchbox/templatev1beta2"
	"github.com/docker/cli/cli/compose/loader"
	"github.com/docker/lunchbox/packager"
	"github.com/docker/lunchbox/templateloader"
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
	res := &helmMeta{
		Name:        meta.Name,
		Version:     meta.Version,
		Description: meta.Description,
	}
	for _, m := range meta.Maintainers {
		res.Maintainers = append(res.Maintainers,
			helmMaintainer{Name: m.Name + " <" + m.Email + ">"},
		)
	}
	return res, nil
}

// toGoTemplate converts $foo and ${foo} into {{.foo}}
func toGoTemplate(template string) (string, error) {
	start := template
	re := regexp.MustCompile(`(^|[^$])\${?([a-zA-Z0-9_.]+)}?`)
	template = re.ReplaceAllString(template, "$1{{.$2}}")
	template = strings.Replace(template, "$$", "$", -1)
	fmt.Printf("%s -> %s\n", start, template)
	return template, nil
}

func convertTemplatesList(list []interface{}) error {
	for i, v := range list {
		switch vv := v.(type) {
			case string:
				vv, err := toGoTemplate(vv)
				if err != nil {
					return err
				}
				list[i] = vv
			case map[interface{}]interface{}:
				err := convertTemplates(vv)
				if err != nil {
					return err
				}
			case []interface{}:
				convertTemplatesList(vv)
		}
	}
	return nil
}

// convertTemplates replaces $foo with {{ .foo }}, and resolves template_ keys
func convertTemplates(dict map[interface{}]interface{}) error {
	for k, v := range dict {
		kk := k.(string)
		if strings.HasPrefix(kk, "template_") {
			dk := strings.TrimPrefix(kk, "template_")
			vd, ok := v.(map[interface{}]interface{})
			if !ok {
				return fmt.Errorf("Expected a map, got %T", v)
			}
			template, ok := vd["valuetemplate"]
			if ok {
				var err error
				template, err = toGoTemplate(template.(string))
				if err != nil {
					return err
				}
			} else {
				template = vd["value"]
			}
			delete(dict, k)
			dict[dk] = template
		} else {
			switch vv := v.(type) {
			case string:
				vv, err := toGoTemplate(vv)
				if err != nil {
					return err
				}
				dict[k] = vv
			case map[interface{}]interface{}:
				err := convertTemplates(vv)
				if err != nil {
					return err
				}
			case []interface{}:
				err := convertTemplatesList(vv)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// Helm renders an app as an Helm Chart
func Helm(appname string, composeFiles []string, settingsFile []string, env map[string]string) error {
	appname, cleanup, err := packager.Extract(appname)
	if err != nil {
		return err
	}
	defer cleanup()
	data, err := ioutil.ReadFile(filepath.Join(appname, "docker-compose.yml"))
	if err != nil {
		return err
	}
	parsed, err := loader.ParseYAML(data)
	rendered, err := templateloader.LoadTemplate(parsed)
	//rendered, err := Render(appname, composeFiles, settingsFile, env)
	if err != nil {
		return err
	}
	metaFile := filepath.Join(appname, "metadata.yml")
	metaContent, err := ioutil.ReadFile(metaFile)
	if err != nil {
		return err
	}
	var meta types.AppMetadata
	err = yaml.Unmarshal(metaContent, &meta)
	if err != nil {
		return err
	}
	targetDir := utils.AppNameFromDir(appname) + ".chart"
	os.Mkdir(targetDir, 0755)
	hmeta, err := toHelmMeta(&meta)
	if err != nil {
		return err
	}
	chart := make(map[interface{}]interface{})
	prevChartRaw, err := ioutil.ReadFile(filepath.Join(targetDir, "Chart.yaml"))
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
	err = ioutil.WriteFile(filepath.Join(targetDir, "Chart.yaml"), hmetadata, 0644)
	if err != nil {
		return err
	}
	os.Mkdir(filepath.Join(targetDir, "templates"), 0755)
	stackSpec := templateconversion.FromComposeConfig(rendered)
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
	preStack := make(map[interface{}]interface{})
	err = yaml.Unmarshal(stackData, preStack)
	if err != nil {
		return err
	}
	err = convertTemplates(preStack)
	if err != nil {
		return err
	}
	stackData, err = yaml.Marshal(preStack)
	err = ioutil.WriteFile(filepath.Join(targetDir, "templates", "stack.yaml"), stackData, 0644)
	return err
}
