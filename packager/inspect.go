package packager

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/docker/lunchbox/types"
	yaml "gopkg.in/yaml.v2"
)

// Inspect dumps the metadata of an app
func Inspect(appname string) error {
	appname, cleanup, err := Extract(appname)
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
	smeta, err := yaml.Marshal(meta)
	if err != nil {
		return err
	}
	fmt.Println(string(smeta))
	return nil
}
