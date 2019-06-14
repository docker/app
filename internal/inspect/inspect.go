package inspect

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/deislabs/cnab-go/bundle"
	"github.com/docker/app/render"
	"github.com/docker/app/types"
	"github.com/docker/app/types/parameters"
	composetypes "github.com/docker/cli/cli/compose/types"
	units "github.com/docker/go-units"
)

// Inspect dumps the metadata of an app
func Inspect(out io.Writer, app *types.App, argParameters map[string]string, imageMap map[string]bundle.Image) error {
	// Render the compose file
	config, err := render.Render(app, argParameters, imageMap)
	if err != nil {
		return err
	}

	// Extract all the parameters
	parametersKeys, allParameters, err := extractParameters(app, argParameters)
	if err != nil {
		return err
	}

	// Add Meta data
	printMetadata(out, app)

	// Add Service section
	printServices(out, config.Services, imageMap)

	// Add Network section
	printSection(out, len(config.Networks), func(w io.Writer) {
		names := make([]string, 0, len(config.Networks))
		for name := range config.Networks {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			fmt.Fprintln(w, name)
		}
	}, "Network")

	// Add Volume section
	printSection(out, len(config.Volumes), func(w io.Writer) {
		names := make([]string, 0, len(config.Volumes))
		for name := range config.Volumes {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			fmt.Fprintln(w, name)
		}
	}, "Volume")

	// Add Secret section
	printSection(out, len(config.Secrets), func(w io.Writer) {
		names := make([]string, 0, len(config.Secrets))
		for name := range config.Secrets {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			fmt.Fprintln(w, name)
		}
	}, "Secret")

	// Add Parameter section
	printSection(out, len(parametersKeys), func(w io.Writer) {
		for _, k := range parametersKeys {
			fmt.Fprintf(w, "%s\t%s\n", k, allParameters[k])
		}
	}, "Parameter", "Value")

	// Add Attachments section
	attachments := app.Attachments()
	printSection(out, len(attachments), func(w io.Writer) {
		for _, file := range attachments {
			sizeString := units.HumanSize(float64(file.Size()))
			fmt.Fprintf(w, "%s\t%s\n", file.Path(), sizeString)
		}
	}, "Attachment", "Size")

	return nil
}

func printServices(out io.Writer, services composetypes.Services, imageMap map[string]bundle.Image) {
	servicesSectionHeader, printService := getServicesPrinter(imageMap)
	printSection(out, len(services), func(w io.Writer) {
		sort.Slice(services, func(i, j int) bool {
			return services[i].Name < services[j].Name
		})
		for _, service := range services {
			printService(w, service)
		}
	}, servicesSectionHeader...)
}

func getServicesPrinter(imageMap map[string]bundle.Image) ([]string, func(io.Writer, composetypes.ServiceConfig)) {
	for _, image := range imageMap {
		if image.BaseImage.OriginalImage != "" && image.BaseImage.OriginalImage != image.BaseImage.Image {
			return []string{"Service", "Replicas", "Ports", "Image", "Reference"}, func(w io.Writer, service composetypes.ServiceConfig) {
				fmt.Fprintf(w, "%s\t%d\t%s\t%s\t%s\n", service.Name, getReplicas(service), getPorts(service.Ports), imageMap[service.Name].OriginalImage, service.Image)
			}
		}
	}
	return []string{"Service", "Replicas", "Ports", "Image"}, func(w io.Writer, service composetypes.ServiceConfig) {
		fmt.Fprintf(w, "%s\t%d\t%s\t%s\n", service.Name, getReplicas(service), getPorts(service.Ports), service.Image)
	}
}

func printMetadata(out io.Writer, app *types.App) {
	meta := app.Metadata()
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
