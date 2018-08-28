package inspect

import (
	"fmt"
	"io"
	"sort"
	"text/tabwriter"

	"github.com/docker/app/types"
)

// Inspect dumps the metadata of an app
func Inspect(out io.Writer, app *types.App) error {
	meta := app.Metadata()
	// extract settings
	settings := app.Settings().Flatten()
	// sort the keys to get consistent output
	var settingsKeys []string
	for k := range settings {
		settingsKeys = append(settingsKeys, k)
	}
	sort.Slice(settingsKeys, func(i, j int) bool { return settingsKeys[i] < settingsKeys[j] })
	// build maintainers string
	maintainers := meta.Maintainers.String()
	fmt.Fprintf(out, "%s %s\n", meta.Name, meta.Version)
	if maintainers != "" {
		fmt.Fprintf(out, "Maintained by: %s\n", maintainers)
		fmt.Fprintln(out, "")
	}
	if meta.Description != "" {
		fmt.Fprintf(out, "%s\n", meta.Description)
		fmt.Fprintln(out, "")
	}
	w := tabwriter.NewWriter(out, 0, 0, 1, ' ', 0)
	fmt.Fprintln(w, "Setting\tDefault")
	fmt.Fprintln(w, "-------\t-------")
	for _, k := range settingsKeys {
		fmt.Fprintf(w, "%s\t%s\n", k, settings[k])
	}
	w.Flush()
	return nil
}
