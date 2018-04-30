package packager

import (
	"archive/tar"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/lunchbox/constants"
	"github.com/docker/lunchbox/utils"
	"github.com/pkg/errors"
)

var (
	noop = func() {}
)

// findApp looks for an app in CWD or subdirs
func findApp() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", errors.Wrap(err, "cannot resolve current working directory")
	}
	if strings.HasSuffix(cwd, constants.AppExtension) {
		return cwd, nil
	}
	content, err := ioutil.ReadDir(cwd)
	if err != nil {
		return "", errors.Wrap(err, "failed to read current working directory")
	}
	hit := ""
	for _, c := range content {
		if strings.HasSuffix(c.Name(), constants.AppExtension) {
			if hit != "" {
				return "", fmt.Errorf("multiple apps found in current directory, specify the app on the command line")
			}
			hit = c.Name()
		}
	}
	if hit == "" {
		return "", fmt.Errorf("no app found in current directory")
	}
	return filepath.Join(cwd, hit), nil
}

// Extract extracts the app content if argument is an archive, or does nothing if a dir.
// It returns effective app name, and cleanup function
// If appname is empty, it looks into cwd, and all subdirs for a single matching .dockerapp
func Extract(appname string) (string, func(), error) {
	if appname == "" {
		var err error
		if appname, err = findApp(); err != nil {
			return "", nil, err
		}
	}
	if appname == "." {
		var err error
		if appname, err = os.Getwd(); err != nil {
			return "", nil, errors.Wrap(err, "cannot resolve current working directory")
		}
	}
	// try verbatim first
	s, err := os.Stat(appname)
	if err != nil {
		// try appending our extension
		appname = utils.DirNameFromAppName(appname)
		s, err = os.Stat(appname)
	}
	if err != nil {
		return "", noop, err
	}
	if s.IsDir() {
		// directory: already decompressed
		return appname, noop, nil
	}
	// not a dir: probably a tarball package, extract that in a temp dir
	tempDir, err := ioutil.TempDir("", "dockerapp")
	if err != nil {
		return "", noop, err
	}
	appDir := filepath.Join(tempDir, filepath.Base(appname))
	if err := os.Mkdir(appDir, 0755); err != nil {
		os.RemoveAll(tempDir)
		return "", noop, err
	}
	if err = extract(appname, appDir); err != nil {
		os.RemoveAll(tempDir)
		return "", noop, err
	}
	return appDir, func() { os.RemoveAll(tempDir) }, nil
}

func extract(appname, outputDir string) error {
	f, err := os.Open(appname)
	if err != nil {
		return err
	}
	tarReader := tar.NewReader(f)
	outputDir = outputDir + "/"
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return errors.Wrap(err, "error reading from tar header")
		}
		switch header.Typeflag {
		case tar.TypeDir: // = directory
			os.Mkdir(outputDir+header.Name, 0755)
		case tar.TypeReg: // = regular file
			data := make([]byte, header.Size)
			_, err := tarReader.Read(data)
			if err != nil && err != io.EOF {
				return errors.Wrap(err, "error reading from tar data")
			}
			err = ioutil.WriteFile(outputDir+header.Name, data, 0644)
			if err != nil {
				return errors.Wrap(err, "error writing output file")
			}
		}
	}
	return nil
}
