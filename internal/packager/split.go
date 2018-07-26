package packager

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/docker/app/internal"
)

// Split converts an app package to the split version
func Split(appname string, outputDir string) error {
	err := os.Mkdir(outputDir, 0755)
	if err != nil {
		return err
	}
	for _, n := range internal.FileNames {
		input, err := ioutil.ReadFile(filepath.Join(appname, n))
		if err != nil {
			return err
		}
		err = ioutil.WriteFile(filepath.Join(outputDir, n), input, 0644)
		if err != nil {
			return err
		}
	}
	return nil
}

// Merge converts an app-package to the single-file merged version
func Merge(appname string, target io.Writer) error {
	for i, n := range internal.FileNames {
		input, err := ioutil.ReadFile(filepath.Join(appname, n))
		if err != nil {
			return err
		}
		if _, err := target.Write(input); err != nil {
			return err
		}
		if i != 2 {
			if _, err := io.WriteString(target, "\n---\n"); err != nil {
				return err
			}
		}
	}
	return nil
}
