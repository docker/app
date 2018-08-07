package inspect

import (
	"fmt"
	"io"
	"sort"
	"text/tabwriter"

	"github.com/docker/app/internal/settings"
	"github.com/docker/app/types"
	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"
)

// Inspect dumps the metadata of an app
func Inspect(out io.Writer, app *types.App) error {
	var meta types.AppMetadata
	err := yaml.Unmarshal(app.Metadata(), &meta)
	if err != nil {
		return errors.Wrap(err, "failed to parse application metadata")
	}
	// extract settings
	s, err := settings.LoadMultiple(app.Settings())
	if err != nil {
		return errors.Wrap(err, "failed to load application settings")
	}
	fs := s.Flatten()
	// sort the keys to get consistent output
	var settingsKeys []string
	for k := range fs {
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
		fmt.Fprintf(w, "%s\t%s\n", k, fs[k])
	}
	w.Flush()
	return nil
}
