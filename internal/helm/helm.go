package helm

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/docker/app/internal"
	"github.com/docker/app/internal/compose"
	"github.com/docker/app/internal/helm/templateconversion"
	"github.com/docker/app/internal/helm/templateloader"
	"github.com/docker/app/internal/helm/templatetypes"
	"github.com/docker/app/internal/helm/templatev1beta2"
	"github.com/docker/app/internal/slices"
	"github.com/docker/app/internal/yaml"
	"github.com/docker/app/render"
	"github.com/docker/app/types"
	"github.com/docker/app/types/metadata"
	"github.com/docker/app/types/settings"
	"github.com/docker/cli/cli/command/stack/kubernetes"
	"github.com/docker/cli/cli/compose/loader"
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

// v1beta1StackSpec is a copy of v1beta1.StackSpec with the proper YAML annotations
type v1beta1StackSpec struct {
	ComposeFile string `json:"composeFile,omitempty" yaml:"composeFile,omitempty"`
}

// v1beta1Stack is a copy of v1beta1.Stack with the proper YAML annotations
type v1beta1Stack struct {
	templatev1beta2.TypeMeta `yaml:",inline" json:",inline"`
	metav1.ObjectMeta        `json:"metadata,omitempty" yaml:"metadata,omitempty"`

	Spec   v1beta1StackSpec    `json:"spec,omitempty" yaml:"spec,omitempty"`
	Status v1beta1.StackStatus `json:"status,omitempty" yaml:"status,omitempty"`
}

// v1beta2Stack is a copy of v1beta2.Stack with the proper YAML annotations
type v1beta2Stack struct {
	templatev1beta2.TypeMeta `json:",inline" yaml:",inline"`
	metav1.ObjectMeta        `json:"metadata,omitempty" yaml:"metadata,omitempty"`

	Spec   *v1beta2.StackSpec   `json:"spec,omitempty" yaml:"spec,omitempty"`
	Status *v1beta2.StackStatus `json:"status,omitempty" yaml:"status,omitempty"`
}

// Helm renders an app as an Helm Chart
func Helm(app *types.App, env map[string]string, shouldRender bool, stackVersion string) error {
	targetDir := internal.AppNameFromDir(app.Name) + ".chart"
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return errors.Wrap(err, "failed to create Chart directory")
	}
	meta := app.Metadata()
	if err := makeChart(&meta, targetDir); err != nil {
		return err
	}
	if shouldRender {
		return helmRender(app, targetDir, env, stackVersion)
	}
	// FIXME(vdemeester) support multiple file for helm
	if len(app.Composes()) > 1 {
		return errors.New("helm rendering doesn't support multiple composefiles")
	}
	data := app.Composes()[0]
	// FIXME(vdemeester): remove the need to create this slice
	variables := []string{}
	vars, err := compose.ExtractVariables(data, render.Pattern)
	if err != nil {
		return err
	}
	for k := range vars {
		variables = append(variables, k)
	}
	if err := makeStack(app.Name, targetDir, data, stackVersion); err != nil {
		return err
	}
	return makeValues(app, targetDir, env, variables)
}

// makeValues updates helm values.yaml with used variables from settings and env
func makeValues(app *types.App, targetDir string, env map[string]string, variables []string) error {
	// merge our variables into Values.yaml
	s := app.Settings()
	metaPrefixed, err := settings.Load(app.MetadataRaw(), settings.WithPrefix("app"))
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
	var stackData []byte
	switch stackVersion {
	case V1Beta2:
		stackSpec := templateconversion.FromComposeConfig(rendered)
		stack := templatev1beta2.Stack{
			TypeMeta:   typeMeta(stackVersion),
			ObjectMeta: objectMeta(appname),
			Spec:       stackSpec,
		}
		templatetypes.ProcessTemplate = toGoTemplate
		stackData, err = yaml.Marshal(stack)
		if err != nil {
			return err
		}
	case V1Beta1:
		templatetypes.ProcessTemplate = toGoTemplate
		composeFile, err := yaml.Marshal(rendered)
		if err != nil {
			return err
		}
		stack := v1beta1Stack{
			TypeMeta:   typeMeta(stackVersion),
			ObjectMeta: objectMeta(appname),
			Spec: v1beta1StackSpec{
				ComposeFile: string(composeFile),
			},
		}
		stackData, err = yaml.Marshal(stack)
		if err != nil {
			return errors.Wrap(err, "failed to marshal final stack")
		}
	default:
		return fmt.Errorf("invalid stack version %q", stackVersion)
	}
	stackData = unquote(stackData)
	return ioutil.WriteFile(filepath.Join(targetDir, "templates", "stack.yaml"), stackData, 0644)
}

func helmRender(app *types.App, targetDir string, env map[string]string, stackVersion string) error {
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
		stack = v1beta2Stack{
			TypeMeta:   typeMeta(stackVersion),
			ObjectMeta: objectMeta(app.Path),
			Spec:       s.Spec,
		}
	case V1Beta1:
		stack = v1beta1Stack{
			TypeMeta:   typeMeta(stackVersion),
			ObjectMeta: objectMeta(app.Path),
			Spec: v1beta1StackSpec{
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

func makeChart(meta *metadata.AppMetadata, targetDir string) error {
	hmeta, err := toHelmMeta(meta)
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

func typeMeta(stackVersion string) templatev1beta2.TypeMeta {
	return templatev1beta2.TypeMeta{
		Kind:       "Stack",
		APIVersion: "compose.docker.com/" + stackVersion,
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

func toHelmMeta(meta *metadata.AppMetadata) (*helmMeta, error) {
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

// unquote unquotes gotemplates in template
func unquote(template []byte) []byte {
	re := regexp.MustCompile(`'(\{\{[^'}]*\}\})'`)
	return re.ReplaceAll(template, []byte("$1"))
}

// toGoTemplate converts $foo and ${foo} into {{.foo}}
func toGoTemplate(template string) (string, error) {
	re := regexp.MustCompile(`(^|[^$])\${?([a-zA-Z0-9_.]+)}?`)
	template = re.ReplaceAllString(template, "$1{{.Values.$2}}")
	template = strings.Replace(template, "$$", "$", -1)
	return template, nil
}
