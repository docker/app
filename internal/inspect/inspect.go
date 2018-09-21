package inspect

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/docker/app/render"
	"github.com/docker/app/types"
	"github.com/docker/app/types/settings"
	composetypes "github.com/docker/cli/cli/compose/types"
)

// Inspect dumps the metadata of an app
func Inspect(out io.Writer, app *types.App, argSettings map[string]string) error {
	// Render the compose file
	config, err := render.Render(app, argSettings)
	if err != nil {
		return err
	}

	// Extract all the settings
	settingsKeys, allSettings, err := extractSettings(app, argSettings)
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
	printSection(out, len(settingsKeys), func(w io.Writer) {
		for _, k := range settingsKeys {
			fmt.Fprintf(w, "%s\t%s\n", k, allSettings[k])
		}
	}, "Setting", "Value")

	// Add External Files section
	externalFiles := app.ExternalFilePaths()
	printSection(out, len(externalFiles), func(w io.Writer) {
		for _, name := range externalFiles {
			fmt.Fprintln(w, name) // and info.Size() ?
		}
	}, "External File")

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

func extractSettings(app *types.App, argSettings map[string]string) ([]string, map[string]string, error) {
	allSettings, err := mergeAndFlattenSettings(app, argSettings)
	if err != nil {
		return nil, nil, err
	}
	// sort the keys to get consistent output
	var settingsKeys []string
	for k := range allSettings {
		settingsKeys = append(settingsKeys, k)
	}
	sort.Slice(settingsKeys, func(i, j int) bool { return settingsKeys[i] < settingsKeys[j] })
	return settingsKeys, allSettings, nil
}

func mergeAndFlattenSettings(app *types.App, argSettings map[string]string) (map[string]string, error) {
	sArgs, err := settings.FromFlatten(argSettings)
	if err != nil {
		return nil, err
	}
	s, err := settings.Merge(app.Settings(), sArgs)
	if err != nil {
		return nil, err
	}
	return s.Flatten(), nil
}
