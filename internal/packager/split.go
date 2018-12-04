package packager

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/docker/app/internal"
	"github.com/docker/app/types"
	"github.com/pkg/errors"
)

// Split converts an app package to the split version
func Split(app *types.App, outputDir string) error {
	if len(app.Composes()) > 1 {
		return errors.New("split: multiple compose files is not supported")
	}
	if len(app.ParametersRaw()) > 1 {
		return errors.New("split: multiple setting files is not supported")
	}
	err := os.MkdirAll(outputDir, 0755)
	if err != nil {
		return err
	}
	for file, data := range map[string][]byte{
		internal.MetadataFileName:   app.MetadataRaw(),
		internal.ComposeFileName:    app.Composes()[0],
		internal.ParametersFileName: app.ParametersRaw()[0],
	} {
		if err := ioutil.WriteFile(filepath.Join(outputDir, file), data, 0644); err != nil {
			return err
		}
	}
	return nil
}

// Merge converts an app-package to the single-file merged version
func Merge(app *types.App, target io.Writer) error {
	if len(app.Composes()) > 1 {
		return errors.New("merge: multiple compose files is not supported")
	}
	if len(app.ParametersRaw()) > 1 {
		return errors.New("merge: multiple setting files is not supported")
	}
	for _, data := range [][]byte{
		app.MetadataRaw(),
		[]byte(types.SingleFileSeparator),
		app.Composes()[0],
		[]byte(types.SingleFileSeparator),
		app.ParametersRaw()[0],
	} {
		if _, err := target.Write(data); err != nil {
			return err
		}
	}
	return nil
}
