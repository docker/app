package renderer

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/docker/lunchbox/packager"
	"github.com/docker/lunchbox/types"
	yaml "gopkg.in/yaml.v2"
)

// Inspect dumps the metadata of an app
func Inspect(appname string) error {
	appname, cleanup, err := packager.Extract(appname)
	if err != nil {
		return err
	}
	defer cleanup()
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
	// extract settings
	settingsFile := filepath.Join(appname, "settings.yml")
	settingsContent, err := ioutil.ReadFile(settingsFile)
	if err != nil {
		return err
	}
	settings, err := flattenYAML(settingsContent)
	if err != nil {
		return err
	}
	// build maintainers string
	maintainers := ""
	for _, m := range meta.Maintainers {
		maintainers += m.Name + " <" + m.Email + ">, "
	}
	maintainers = strings.TrimSuffix(maintainers, ", ")
	fmt.Printf("%s %s\n", meta.Name, meta.Version)
	fmt.Printf("Maintained by: %s\n", maintainers)
	fmt.Println("")
	fmt.Printf("%s\n", meta.Description)
	fmt.Println("")
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
	fmt.Fprintln(w, "Setting\tDefault")
	fmt.Fprintln(w, "-------\t-------")
	for k, v := range settings {
		fmt.Fprintf(w, "%s\t%s\n", k, v)
	}
	w.Flush()
	return nil
}
