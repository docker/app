package inspect

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/deislabs/cnab-go/bundle"
	"github.com/docker/app/internal"
	"github.com/docker/app/render"
	"github.com/docker/app/types"
	"github.com/docker/app/types/metadata"
	"github.com/docker/app/types/parameters"
	composetypes "github.com/docker/cli/cli/compose/types"
	units "github.com/docker/go-units"
)

type service struct {
	Name     string `json:",omitempty"`
	Image    string `json:",omitempty"`
	Replicas int    `json:",omitempty"`
	Mode     string `json:",omitempty"`
	Ports    string `json:",omitempty"`
}

type attachment struct {
	Path string `json:",omitempty"`
	Size int64  `json:",omitempty"`
}

type appInfo struct {
	Metadata       metadata.AppMetadata `json:",omitempty"`
	Services       []service            `json:",omitempty"`
	Networks       []string             `json:",omitempty"`
	Volumes        []string             `json:",omitempty"`
	Secrets        []string             `json:",omitempty"`
	parametersKeys []string
	Parameters     map[string]string `json:",omitempty"`
	Attachments    []attachment      `json:",omitempty"`
}

// Inspect dumps the metadata of an app
func Inspect(out io.Writer, app *types.App, argParameters map[string]string, imageMap map[string]bundle.Image) error {
	// Render the compose file
	config, err := render.Render(app, argParameters, imageMap)
	if err != nil {
		return err
	}

	// Collect all the relevant information about the application
	appInfo, err := getAppInfo(app, config, argParameters)
	if err != nil {
		return err
	}

	outputFormat := os.Getenv(internal.DockerInspectFormatEnvVar)
	return printAppInfo(out, appInfo, outputFormat)
}

func printAppInfo(out io.Writer, app appInfo, format string) error {
	switch format {
	case "pretty":
		return printTable(out, app)
	case "json":
		return printJSON(out, app)
	default:
		return fmt.Errorf("unknown format %q", format)
	}
}

func printJSON(out io.Writer, appInfo appInfo) error {
	js, err := json.MarshalIndent(appInfo, "", "    ")
	if err != nil {
		return err
	}
	fmt.Fprintln(out, string(js))
	return nil
}

func printTable(out io.Writer, appInfo appInfo) error {
	// Add Meta data
	printMetadata(out, appInfo)

	// Add Service section
	printSection(out, len(appInfo.Services), func(w io.Writer) {
		for _, service := range appInfo.Services {
			fmt.Fprintf(w, "%s\t%d\t%s\t%s\n", service.Name, service.Replicas, service.Ports, service.Image)
		}
	}, "Service", "Replicas", "Ports", "Image")

	// Add Network section
	printSection(out, len(appInfo.Networks), func(w io.Writer) {
		for _, name := range appInfo.Networks {
			fmt.Fprintln(w, name)
		}
	}, "Network")

	// Add Volume section
	printSection(out, len(appInfo.Volumes), func(w io.Writer) {
		for _, name := range appInfo.Volumes {
			fmt.Fprintln(w, name)
		}
	}, "Volume")

	// Add Secret section
	printSection(out, len(appInfo.Secrets), func(w io.Writer) {
		for _, name := range appInfo.Secrets {
			fmt.Fprintln(w, name)
		}
	}, "Secret")

	// Add Parameter section
	printSection(out, len(appInfo.parametersKeys), func(w io.Writer) {
		for _, k := range appInfo.parametersKeys {
			fmt.Fprintf(w, "%s\t%s\n", k, appInfo.Parameters[k])
		}
	}, "Parameter", "Value")

	// Add Attachments section
	printSection(out, len(appInfo.Attachments), func(w io.Writer) {
		for _, attachment := range appInfo.Attachments {
			sizeString := units.HumanSize(float64(attachment.Size))
			fmt.Fprintf(w, "%s\t%s\n", attachment.Path, sizeString)
		}
	}, "Attachment", "Size")
	return nil
}

func printMetadata(out io.Writer, app appInfo) {
	meta := app.Metadata
	fmt.Fprintln(out, meta.Name, meta.Version)
	if maintainers := meta.Maintainers.String(); maintainers != "" {
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Maintained by:", maintainers)
	}
	if meta.Description != "" {
		fmt.Fprintln(out)
		fmt.Fprintln(out, meta.Description)
	}
}

func printSection(out io.Writer, len int, printer func(io.Writer), headers ...string) {
	if len == 0 {
		return
	}
	fmt.Fprintln(out)
	w := tabwriter.NewWriter(out, 0, 0, 1, ' ', 0)
	var plural string
	if len > 1 {
		plural = "s"
	}
	headers[0] = fmt.Sprintf("%s%s (%d)", headers[0], plural, len)
	printHeaders(w, headers...)
	printer(w)
	w.Flush()
}

func printHeaders(w io.Writer, headers ...string) {
	fmt.Fprintln(w, strings.Join(headers, "\t"))
	dashes := make([]string, len(headers))
	for i, h := range headers {
		dashes[i] = strings.Repeat("-", len(h))
	}
	fmt.Fprintln(w, strings.Join(dashes, "\t"))
}

func getAppInfo(app *types.App, config *composetypes.Config, argParameters map[string]string) (appInfo, error) {
	services := []service{}
	for _, s := range config.Services {
		services = append(services, service{
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
		return appInfo{}, err
	}

	attachments := []attachment{}
	appAttachments := app.Attachments()
	for _, file := range appAttachments {
		attachments = append(attachments, attachment{
			Path: file.Path(),
			Size: file.Size(),
		})
	}

	return appInfo{
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
	sort.Slice(parametersKeys, func(i, j int) bool { return parametersKeys[i] < parametersKeys[j] })
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
