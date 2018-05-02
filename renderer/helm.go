package renderer

import (
	"io/ioutil"
	"os"
	"path/filepath"

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

// Helm renders an app as an Helm Chart
func Helm(appname string, composeFiles []string, settingsFile []string, env map[string]string) error {
	appname, cleanup, err := packager.Extract(appname)
	if err != nil {
		return err
	}
	defer cleanup()
	rendered, err := Render(appname, composeFiles, settingsFile, env)
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
	stackSpec := conversion.FromComposeConfig(rendered)
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
	err = ioutil.WriteFile(filepath.Join(targetDir, "templates", "stack.yaml"), stackData, 0644)
	return err
}
