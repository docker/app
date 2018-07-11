package render

import (
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"
	"sort"
	"text/tabwriter"

	"github.com/docker/app/internal"
	"github.com/docker/app/internal/types"
	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"
)

// Inspect dumps the metadata of an app
func Inspect(out io.Writer, appname string) error {
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
	// extract settings
	settingsFile := filepath.Join(appname, internal.SettingsFileName)
	settingsContent, err := ioutil.ReadFile(settingsFile)
	if err != nil {
		return errors.Wrap(err, "failed to read application settings")
	}
	settings, err := flattenYAML(settingsContent)
	if err != nil {
		return errors.Wrap(err, "failed to parse application settings")
	}
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
