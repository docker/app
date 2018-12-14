package inspect

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/deislabs/duffle/pkg/bundle"
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
	printSection(out, len(config.Services), func(w io.Writer) {
		for _, service := range config.Services {
			fmt.Fprintf(w, "%s\t%d\t%s\t%s\n", service.Name, getReplicas(service), getPorts(service.Ports), service.Image)
		}
	}, "Service", "Replicas", "Ports", "Image")

	// Add Network section
	printSection(out, len(config.Networks), func(w io.Writer) {
		for name := range config.Networks {
			fmt.Fprintln(w, name)
		}
	}, "Network")

	// Add Volume section
	printSection(out, len(config.Volumes), func(w io.Writer) {
		for name := range config.Volumes {
			fmt.Fprintln(w, name)
		}
	}, "Volume")

	// Add Secret section
	printSection(out, len(config.Secrets), func(w io.Writer) {
		for name := range config.Secrets {
			fmt.Fprintln(w, name)
		}
	}, "Secret")

	// Add Setting section
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
