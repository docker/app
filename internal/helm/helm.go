package helm

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	yaml "gopkg.in/yaml.v2"

	"github.com/docker/app/internal"
	"github.com/docker/app/internal/helm/templateconversion"
	"github.com/docker/app/internal/helm/templateloader"
	"github.com/docker/app/internal/helm/templatev1beta2"
	"github.com/docker/app/internal/render"
	"github.com/docker/app/internal/settings"
	"github.com/docker/app/internal/slices"
	"github.com/docker/app/internal/types"
	"github.com/docker/cli/cli/command/stack/kubernetes"
	"github.com/docker/cli/cli/compose/loader"
	"github.com/docker/cli/cli/compose/template"
	"github.com/docker/cli/kubernetes/compose/v1beta1"
	"github.com/docker/cli/kubernetes/compose/v1beta2"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

/* Helm rendering with template preservation.

We modify compose.Type (in templatetypes) by replacing all bool by BoolOrTemplate,
all *uint64 with UInt64OrTemplate, etc.  so that we can store both a value or
a templated string.
We modify compose.Loader (in templateloader) to provide a new LoadTemplate that
skips schema validation and variable interpolation. MapStructure hooks are
provided for our *OrTemplate structs.
We modify v1beta2 Stack and associated structures (in templatev1beta2) in sync
with the changes in compose.Type, with the addition that all *OrTemplate structs
are yaml-serialized with a name prefied by 'template_'.
This package then invokes LoadTemplate, then templatev1beta2.convert, and
post-process the serialized yaml to replace all 'template_'-prefixed keys
with the appropriate content (value or template)
*/

// Helm renders an app as an Helm Chart
func Helm(app types.App, env map[string]string, shouldRender bool, stackVersion string) error {
	targetDir := internal.AppNameFromDir(app.Path) + ".chart"
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return errors.Wrap(err, "failed to create Chart directory")
	}
	err := makeChart(app.Path, targetDir)
	if err != nil {
		return err
	}
	if shouldRender {
		return helmRender(app, targetDir, env, stackVersion)
	}
	// FIXME(vdemeester) handle that
	data, err := ioutil.ReadFile(filepath.Join(app.Path, internal.ComposeFileName))
	if err != nil {
		return errors.Wrap(err, "failed to read application Compose file")
	}
	cfgMap, err := loader.ParseYAML(data)
	if err != nil {
		return errors.Wrap(err, "failed to parse compose file")
	}
	vars := template.ExtractVariables(cfgMap, render.Pattern)
	// FIXME(vdemeester): remove the need to create this slice
	variables := []string{}
	for k := range vars {
		variables = append(variables, k)
	}
	err = makeStack(app.Path, targetDir, data, stackVersion)
	if err != nil {
		return err
	}
	return makeValues(app.Path, targetDir, app.SettingsFiles, env, variables)
}

// makeValues updates helm values.yaml with used variables from settings and env
func makeValues(appname, targetDir string, settingsFile []string, env map[string]string, variables []string) error {
	// merge our variables into Values.yaml
	sf := []string{filepath.Join(appname, internal.SettingsFileName)}
	sf = append(sf, settingsFile...)
	s, err := settings.LoadFiles(sf)
	if err != nil {
		return err
	}
	metaFile := filepath.Join(appname, internal.MetadataFileName)
	metaPrefixed, err := settings.LoadFile(metaFile, settings.WithPrefix("app"))
	if err != nil {
		return err
	}
	envSettings, err := settings.FromFlatten(env)
	if err != nil {
		return err
	}
	s, err = settings.Merge(s, metaPrefixed, envSettings)
	if err != nil {
		return errors.Wrap(err, "failed to merge settings")
	}
	filterVariables(s, variables, "")
	// merge settings with existing values.yml
	values := make(map[interface{}]interface{})
	if valuesCur, err := ioutil.ReadFile(filepath.Join(targetDir, "values.yaml")); err == nil {
		err = yaml.Unmarshal(valuesCur, values)
		if err != nil {
			return errors.Wrap(err, "failed to parse existing values.yaml")
		}
	}
	mergeValues(values, s)
	valuesRaw, err := yaml.Marshal(values)
	if err != nil {
		return errors.Wrap(err, "failed to generate values.yaml")
	}
	return ioutil.WriteFile(filepath.Join(targetDir, "values.yaml"), valuesRaw, 0644)
}

// makeStack converts data into a helm template for a stack
func makeStack(appname string, targetDir string, data []byte, stackVersion string) error {
	parsed, err := loader.ParseYAML(data)
	if err != nil {
		return errors.Wrap(err, "failed to parse template compose")
	}
	rendered, err := templateloader.LoadTemplate(parsed)
	if err != nil {
		return errors.Wrap(err, "failed to load template compose")
	}
	if err := os.MkdirAll(filepath.Join(targetDir, "templates"), 0755); err != nil {
		return err
	}
	var stack interface{}
	switch stackVersion {
	case V1Beta2:
		stackSpec := templateconversion.FromComposeConfig(rendered)
		stack = templatev1beta2.Stack{
			TypeMeta:   typeMeta(stackVersion),
			ObjectMeta: objectMeta(appname),
			Spec:       stackSpec,
		}
	case V1Beta1:
		composeFile, err := yaml.Marshal(rendered)
		if err != nil {
			return err
		}
		stack = v1beta1.Stack{
			TypeMeta:   typeMeta(stackVersion),
			ObjectMeta: objectMeta(appname),
			Spec: v1beta1.StackSpec{
				ComposeFile: string(composeFile),
			},
		}
	default:
		return fmt.Errorf("invalid stack version %q", stackVersion)
	}
	stackData, err := yaml.Marshal(stack)
	if err != nil {
		return errors.Wrap(err, "failed to marshal stack data")
	}
	preStack := make(map[interface{}]interface{})
	err = yaml.Unmarshal(stackData, preStack)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal stack data")
	}
	err = convertTemplates(preStack)
	if err != nil {
		return errors.Wrap(err, "failed to convert stack templates")
	}
	stackData, err = yaml.Marshal(preStack)
	if err != nil {
		return errors.Wrap(err, "failed to marshal final stack")
	}
	return ioutil.WriteFile(filepath.Join(targetDir, "templates", "stack.yaml"), stackData, 0644)
}

func helmRender(app types.App, targetDir string, env map[string]string, stackVersion string) error {
	rendered, err := render.Render(app, env)
	if err != nil {
		return err
	}
	converter, err := kubernetes.NewStackConverter(stackVersion)
	if err != nil {
		return err
	}
	name := internal.AppNameFromDir(app.Path)
	s, err := converter.FromCompose(ioutil.Discard, name, rendered)
	if err != nil {
		return err
	}
	var stack interface{}
	switch stackVersion {
	case V1Beta2:
		stack = v1beta2.Stack{
			TypeMeta:   typeMeta(stackVersion),
			ObjectMeta: objectMeta(app.Path),
			Spec:       s.Spec,
		}
	case V1Beta1:
		stack = v1beta1.Stack{
			TypeMeta:   typeMeta(stackVersion),
			ObjectMeta: objectMeta(app.Path),
			Spec: v1beta1.StackSpec{
				ComposeFile: s.ComposeFile,
			},
		}
	default:
		return fmt.Errorf("invalid stack version %q", stackVersion)
	}
	stackData, err := yaml.Marshal(stack)
	if err != nil {
		return errors.Wrap(err, "failed to marshal stack data")
	}
	return ioutil.WriteFile(filepath.Join(targetDir, "templates", "stack.yaml"), stackData, 0644)
}

func makeChart(appname, targetDir string) error {
	metaFile := filepath.Join(appname, internal.MetadataFileName)
	metaContent, err := ioutil.ReadFile(metaFile)
	if err != nil {
		return errors.Wrap(err, "failed to read application metadata")
	}
	var meta types.AppMetadata
	err = yaml.Unmarshal(metaContent, &meta)
	if err != nil {
		return errors.Wrap(err, "failed to parse application metadata")
	}
	hmeta, err := toHelmMeta(&meta)
	if err != nil {
		return errors.Wrap(err, "failed to convert application metadata")
	}
	chart := make(map[interface{}]interface{})
	prevChartRaw, err := ioutil.ReadFile(filepath.Join(targetDir, "Chart.yaml"))
	if err == nil {
		err = yaml.Unmarshal(prevChartRaw, chart)
		if err != nil {
			return errors.Wrap(err, "failed to unmarshal current Chart.yaml")
		}
	}
	chart["name"] = hmeta.Name
	chart["version"] = hmeta.Version
	chart["description"] = hmeta.Description
	chart["keywords"] = hmeta.Keywords
	chart["maintainers"] = hmeta.Maintainers
	hmetadata, err := yaml.Marshal(chart)
	if err != nil {
		return errors.Wrap(err, "failed to marshal Chart")
	}
	return ioutil.WriteFile(filepath.Join(targetDir, "Chart.yaml"), hmetadata, 0644)
}

func typeMeta(stackVersion string) metav1.TypeMeta {
	return metav1.TypeMeta{
		Kind:       "stacks.compose.docker.com",
		APIVersion: stackVersion,
	}
}

func objectMeta(appname string) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name: internal.AppNameFromDir(appname),
	}
}

const (
	// V1Beta1 is the string identifier for the v1beta1 version of the stack spec
	V1Beta1 = "v1beta1"
	// V1Beta2 is the string identifier for the v1beta2 version of the stack spec
	V1Beta2 = "v1beta2"
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

// remove from settings all stuff that is not in variables
func filterVariables(s map[string]interface{}, variables []string, prefix string) {
	for k, v := range s {
		switch vv := v.(type) {
		case map[string]interface{}:
			filterVariables(vv, variables, prefix+k+".")
			if len(vv) == 0 {
				delete(s, k)
			}
		default:
			if !slices.ContainsString(variables, prefix+k) {
				delete(s, k)
			}
		}
	}
}

// toGoTemplate converts $foo and ${foo} into {{.foo}}
func toGoTemplate(template string) (string, error) {
	re := regexp.MustCompile(`(^|[^$])\${?([a-zA-Z0-9_.]+)}?`)
	template = re.ReplaceAllString(template, "$1{{.Values.$2}}")
	template = strings.Replace(template, "$$", "$", -1)
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
			if err := convertTemplatesList(vv); err != nil {
				return err
			}
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
