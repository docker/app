package packager

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
)

// Split converts an app package to the split version
func Split(appname string, outputDir string) error {
	err := os.Mkdir(outputDir, 0755)
	if err != nil {
		return err
	}
	names := []string{"metadata.yml", "docker-compose.yml", "settings.yml"}
	for _, n := range names {
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
	names := []string{"metadata.yml", "docker-compose.yml", "settings.yml"}
	for i, n := range names {
		input, err := ioutil.ReadFile(filepath.Join(appname, n))
		if err != nil {
			return err
		}
		target.Write(input)
		if i != 2 {
			io.WriteString(target, "\n--\n")
		}
	}
	return nil
}
