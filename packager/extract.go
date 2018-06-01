package packager

import (
	"archive/tar"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/app/constants"
	"github.com/docker/app/utils"
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

// extractImage extracts a docker application in a docker image to a temporary directory
func extractImage(appname string) (string, func(), error) {
	var imagename string
	if strings.Contains(appname, ":") {
		nametag := strings.SplitN(appname, ":", 2)
		nametag[0] = utils.DirNameFromAppName(nametag[0])
		appname = filepath.Base(nametag[0])
		imagename = strings.Join(nametag, ":")
	} else {
		imagename = utils.DirNameFromAppName(appname)
		appname = filepath.Base(imagename)
	}
	tempDir, err := ioutil.TempDir("", "dockerapp")
	if err != nil {
		return "", noop, err
	}
	defer os.RemoveAll(tempDir)
	err = Load(imagename, tempDir)
	if err != nil {
		return "", noop, fmt.Errorf("could not locate application in either filesystem or docker image")
	}
	// this gave us a compressed app, run through extract again
	return Extract(filepath.Join(tempDir, appname))
}

// Extract extracts the app content if argument is an archive, or does nothing if a dir.
// It returns effective app name, and cleanup function
// If appname is empty, it looks into cwd, and all subdirs for a single matching .dockerapp
// If nothing is found, it looks for an image and loads it
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
	originalAppname := appname
	// try verbatim first
	s, err := os.Stat(appname)
	if err != nil {
		// try appending our extension
		appname = utils.DirNameFromAppName(appname)
		s, err = os.Stat(appname)
	}
	if err != nil {
		// look for a docker image
		return extractImage(originalAppname)
	}
	if s.IsDir() {
		// directory: already decompressed
		return appname, noop, nil
	}
	// not a dir: single-file or a tarball package, extract that in a temp dir
	tempDir, err := ioutil.TempDir("", "dockerapp")
	if err != nil {
		return "", noop, err
	}
	defer func() {
		if err != nil {
			os.RemoveAll(tempDir)
		}
	}()
	appDir := filepath.Join(tempDir, filepath.Base(appname))
	if err = os.Mkdir(appDir, 0755); err != nil {
		return "", noop, err
	}
	if err = extract(appname, appDir); err == nil {
		return appDir, func() { os.RemoveAll(tempDir) }, nil
	}
	if err = extractSingleFile(appname, appDir); err != nil {
		return "", noop, err
	}
	// not a tarball, single-file then
	return appDir, func() { os.RemoveAll(tempDir) }, nil
}

func extractSingleFile(appname, appDir string) error {
	// not a tarball, single-file then
	data, err := ioutil.ReadFile(appname)
	if err != nil {
		return err
	}
	parts := strings.Split(string(data), "\n--")
	if len(parts) != 3 {
		return fmt.Errorf("malformed single-file application: expected 3 documents")
	}
	names := []string{"metadata.yml", "docker-compose.yml", "settings.yml"}
	for i, p := range parts {
		data := ""
		if i == 0 {
			data = p
		} else {
			d := strings.SplitN(p, "\n", 2)
			if len(d) > 1 {
				data = d[1]
			}
		}
		err = ioutil.WriteFile(filepath.Join(appDir, names[i]), []byte(data), 0644)
		if err != nil {
			return err
		}
	}
	return nil
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
