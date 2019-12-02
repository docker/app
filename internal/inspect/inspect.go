package inspect

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/docker/app/internal"
	"github.com/docker/app/internal/store"
	"github.com/docker/app/render"
	"github.com/docker/app/types"
	"github.com/docker/app/types/metadata"
	"github.com/docker/app/types/parameters"

	"github.com/deislabs/cnab-go/bundle"
	"github.com/deislabs/cnab-go/claim"
	composetypes "github.com/docker/cli/cli/compose/types"
	"github.com/docker/go-units"
	humanize "github.com/dustin/go-humanize"
	"gopkg.in/yaml.v2"
)

type Service struct {
	Name     string `json:",omitempty"`
	Image    string `json:",omitempty"`
	Replicas int    `json:",omitempty"`
	Mode     string `json:",omitempty"`
	Ports    string `json:",omitempty"`
}

type Attachment struct {
	Path string `json:",omitempty"`
	Size int64  `json:",omitempty"`
}

type ImageAppInfo struct {
	Metadata       metadata.AppMetadata `json:",omitempty"`
	Services       []Service            `json:",omitempty"`
	Networks       []string             `json:",omitempty"`
	Volumes        []string             `json:",omitempty"`
	Secrets        []string             `json:",omitempty"`
	parametersKeys []string
	Parameters     map[string]string `json:",omitempty"`
	Attachments    []Attachment      `json:",omitempty"`
}

type Installation struct {
	Name         string `yaml:"Name,omitempty" json:",omitempty"`
	Created      string `yaml:"Created,omitempty" json:",omitempty"`
	Modified     string `yaml:"Modified,omitempty" json:",omitempty"`
	Revision     string `yaml:"Revision,omitempty" json:",omitempty"`
	LastAction   string `yaml:"Last Action,omitempty" json:"Last Action,omitempty"`
	Result       string `yaml:"Result,omitempty" json:",omitempty"`
	Orchestrator string `yaml:"Ochestrator,omitempty" json:",omitempty"`
}

type Application struct {
	Name           string `yaml:"Name,omitempty" json:",omitempty"`
	Version        string `yaml:"Version,omitempty" json:",omitempty"`
	ImageReference string `yaml:"Image Reference,omitempty" json:"ImageReference,omitempty"`
}

type AppInfo struct {
	Installation Installation           `yaml:"Running App,omitempty" json:"RunningApp,omitempty"`
	Application  Application            `yaml:"App,omitempty" json:"App,omitempty"`
	Parameters   map[string]interface{} `yaml:"Parameters,omitempty" json:"Parameters,omitempty"`
}

func Inspect(out io.Writer, installation *store.Installation, outputFormat string) error {
	// Collect all the relevant information about the application
	appInfo := GetAppInfo(installation)
	return printAppInfo(out, appInfo, outputFormat)
}

func GetAppInfo(installation *store.Installation) AppInfo {
	return AppInfo{
		Installation: Installation{
			Name:         installation.Name,
			Created:      humanize.Time(installation.Created),
			Modified:     humanize.Time(installation.Modified),
			Revision:     installation.Revision,
			LastAction:   installation.Result.Action,
			Result:       installation.Result.Status,
			Orchestrator: getOrchestrator(installation.Claim),
		},
		Application: Application{
			Name:           installation.Bundle.Name,
			Version:        installation.Bundle.Version,
			ImageReference: installation.Reference,
		},
		Parameters: removeDockerAppParameters(installation.Parameters),
	}
}

func ImageInspect(out io.Writer, app *types.App, argParameters map[string]string, imageMap map[string]bundle.Image) error {
	// Render the compose file
	config, err := render.Render(app, argParameters, imageMap)
	if err != nil {
		return err
	}

	// Collect all the relevant information about the application
	appInfo, err := getImageAppInfo(app, config, argParameters)
	if err != nil {
		return err
	}

	outputFormat := os.Getenv(internal.DockerInspectFormatEnvVar)
	return printImageAppInfo(out, appInfo, outputFormat, true)
}

func ImageInspectCNAB(out io.Writer, bndl *bundle.Bundle, outputFormat string) error {
	meta := metadata.AppMetadata{
		Description: bndl.Description,
		Name:        bndl.Name,
		Version:     bndl.Version,
		Maintainers: []metadata.Maintainer{},
	}
	for _, m := range bndl.Maintainers {
		meta.Maintainers = append(meta.Maintainers, metadata.Maintainer{
			Name:  m.Name,
			Email: m.Email,
		})
	}

	paramKeys := []string{}
	params := map[string]string{}
	for _, v := range bndl.Parameters {
		paramKeys = append(paramKeys, v.Definition)
		if d, ok := bndl.Definitions[v.Definition]; ok && d.Default != nil {
			params[v.Definition] = fmt.Sprint(d.Default)
		} else {
			params[v.Definition] = ""
		}
	}
	sort.Strings(paramKeys)

	services := []Service{}
	for k, v := range bndl.Images {
		services = append(services, Service{
			Name:  k,
			Image: v.Image,
		})
	}
	sort.SliceStable(services, func(i, j int) bool {
		return services[i].Name < services[j].Name
	})

	appInfo := ImageAppInfo{
		Metadata:       meta,
		parametersKeys: paramKeys,
		Parameters:     params,
		Services:       services,
	}

	return printImageAppInfo(out, appInfo, outputFormat, false)
}

func printAppInfo(out io.Writer, app AppInfo, format string) error {
	switch format {
	case "pretty":
		return printAppTable(out, app)
	case "json":
		return printJSON(out, app)
	default:
		return fmt.Errorf("unknown format %q", format)
	}
}

func printImageAppInfo(out io.Writer, app ImageAppInfo, format string, isApp bool) error {
	switch format {
	case "pretty":
		return printTable(out, app, isApp)
	case "json":
		return printJSON(out, app)
	default:
		return fmt.Errorf("unknown format %q", format)
	}
}

func printJSON(out io.Writer, appInfo interface{}) error {
	js, err := json.MarshalIndent(appInfo, "", "    ")
	if err != nil {
		return err
	}
	fmt.Fprintln(out, string(js))
	return nil
}

func printAppTable(out io.Writer, info AppInfo) error {

	printYAML(out, AppInfo{
		Installation: info.Installation,
		Application:  Application{},
		Parameters:   nil,
	})
	printYAML(out, AppInfo{
		Installation: Installation{},
		Application:  info.Application,
		Parameters:   nil,
	})
	printYAML(out, AppInfo{
		Installation: Installation{},
		Application:  Application{},
		Parameters:   info.Parameters,
	})

	return nil
}

func printTable(out io.Writer, appInfo ImageAppInfo, isApp bool) error {
	// Add Meta data
	printYAML(out, appInfo.Metadata)

	// Add Service section
	if isApp {
		printSection(out, len(appInfo.Services), func(w io.Writer) {
			for _, service := range appInfo.Services {
				fmt.Fprintf(w, "%s\t%d\t%s\t%s\n", service.Name, service.Replicas, service.Ports, service.Image)
			}
		}, "SERVICE", "REPLICAS", "PORTS", "IMAGE")
	} else {
		printSection(out, len(appInfo.Services), func(w io.Writer) {
			for _, service := range appInfo.Services {
				fmt.Fprintf(w, "%s\t%s\n", service.Name, service.Image)
			}
		}, "SERVICE", "IMAGE")
	}

	// Add Network section
	printSection(out, len(appInfo.Networks), func(w io.Writer) {
		for _, name := range appInfo.Networks {
			fmt.Fprintln(w, name)
		}
	}, "NETWORK")

	// Add Volume section
	printSection(out, len(appInfo.Volumes), func(w io.Writer) {
		for _, name := range appInfo.Volumes {
			fmt.Fprintln(w, name)
		}
	}, "VOLUME")

	// Add Secret section
	printSection(out, len(appInfo.Secrets), func(w io.Writer) {
		for _, name := range appInfo.Secrets {
			fmt.Fprintln(w, name)
		}
	}, "SECRET")

	// Add Parameter section
	printSection(out, len(appInfo.parametersKeys), func(w io.Writer) {
		for _, k := range appInfo.parametersKeys {
			fmt.Fprintf(w, "%s\t%s\n", k, appInfo.Parameters[k])
		}
	}, "PARAMETER", "VALUE")

	// Add Attachments section
	printSection(out, len(appInfo.Attachments), func(w io.Writer) {
		for _, attachment := range appInfo.Attachments {
			sizeString := units.HumanSize(float64(attachment.Size))
			fmt.Fprintf(w, "%s\t%s\n", attachment.Path, sizeString)
		}
	}, "ATTACHMENT", "SIZE")
	return nil
}

func printYAML(out io.Writer, info interface{}) {
	if bytes, err := yaml.Marshal(info); err == nil {
		fmt.Fprintln(out, string(bytes))
	}
}

func printSection(out io.Writer, len int, printer func(io.Writer), headers ...string) {
	if len == 0 {
		return
	}
	fmt.Fprintln(out)
	w := tabwriter.NewWriter(out, 20, 1, 3, ' ', 0)
	fmt.Fprintln(w, strings.Join(headers, "\t"))
	printer(w)
	w.Flush()
}

func getOrchestrator(claim claim.Claim) string {
	if orchestrator, ok := claim.Parameters[internal.ParameterOrchestratorName]; ok && orchestrator != nil {
		return orchestrator.(string)
	}
	return ""
}

func removeDockerAppParameters(parameters map[string]interface{}) map[string]interface{} {
	filteredResults := make(map[string]interface{})
	for key, val := range parameters {
		if !strings.HasPrefix(key, "com.docker.app") {
			filteredResults[key] = val
		}
	}
	return filteredResults
}

func getImageAppInfo(app *types.App, config *composetypes.Config, argParameters map[string]string) (ImageAppInfo, error) {
	services := []Service{}
	for _, s := range config.Services {
		services = append(services, Service{
			Name:     s.Name,
			Image:    s.Image,
			Replicas: getReplicas(s),
			Ports:    getPorts(s.Ports),
		})
	}

	sort.SliceStable(services, func(i, j int) bool {
		return services[i].Name < services[j].Name
	})

	networks := []string{}
	for n := range config.Networks {
		networks = append(networks, n)
	}
	sort.Strings(networks)

	volumes := []string{}
	for v := range config.Volumes {
		volumes = append(volumes, v)
	}
	sort.Strings(volumes)

	secrets := []string{}
	for s := range config.Secrets {
		secrets = append(secrets, s)
	}
	sort.Strings(secrets)

	// Extract all the parameters
	parametersKeys, allParameters, err := extractParameters(app, argParameters)
	if err != nil {
		return ImageAppInfo{}, err
	}

	attachments := []Attachment{}
	appAttachments := app.Attachments()
	for _, file := range appAttachments {
		attachments = append(attachments, Attachment{
			Path: file.Path(),
			Size: file.Size(),
		})
	}

	return ImageAppInfo{
		Metadata:       app.Metadata(),
		Services:       services,
		Networks:       networks,
		Volumes:        volumes,
		Secrets:        secrets,
		parametersKeys: parametersKeys,
		Parameters:     allParameters,
		Attachments:    attachments,
	}, nil
}

func getReplicas(service composetypes.ServiceConfig) int {
	if service.Deploy.Replicas != nil {
		return int(*service.Deploy.Replicas)
	}
	return 1
}

func extractParameters(app *types.App, argParameters map[string]string) ([]string, map[string]string, error) {
	allParameters, err := mergeAndFlattenParameters(app, argParameters)
	if err != nil {
		return nil, nil, err
	}
	// sort the keys to get consistent output
	var parametersKeys []string
	for k := range allParameters {
		parametersKeys = append(parametersKeys, k)
	}
	sort.Strings(parametersKeys)
	return parametersKeys, allParameters, nil
}

func mergeAndFlattenParameters(app *types.App, argParameters map[string]string) (map[string]string, error) {
	sArgs, err := parameters.FromFlatten(argParameters)
	if err != nil {
		return nil, err
	}
	s, err := parameters.Merge(app.Parameters(), sArgs)
	if err != nil {
		return nil, err
	}
	return s.Flatten(), nil
}
