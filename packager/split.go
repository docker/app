package packager

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
)

// Split converts an app package to the split version
func Split(appname string, outputDir string) error {
	appname, cleanup, err := Extract(appname)
	if err != nil {
		return err
	}
	defer cleanup()
	err = os.Mkdir(outputDir, 0755)
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
func Merge(appname string, outputFile string) error {
	appname, cleanup, err := Extract(appname)
	if err != nil {
		return err
	}
	defer cleanup()
	var target io.Writer
	if outputFile == "-" {
		target = os.Stdout
	} else {
		target, err = os.Create(outputFile)
		if err != nil {
			return err
		}
		defer target.(io.WriteCloser).Close()
	}
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
