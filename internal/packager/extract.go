package packager

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/docker/app/internal"
	"github.com/docker/docker/pkg/archive"
	"github.com/pkg/errors"
)

// ExtractedApp represents a potentially extracted application package
type ExtractedApp struct {
	OriginalAppName string
	AppName         string
	Cleanup         func()
}

var (
	noop = func() {}
)

// findApp looks for an app in CWD or subdirs
func findApp() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", errors.Wrap(err, "cannot resolve current working directory")
	}
	if strings.HasSuffix(cwd, internal.AppExtension) {
		return cwd, nil
	}
	content, err := ioutil.ReadDir(cwd)
	if err != nil {
		return "", errors.Wrap(err, "failed to read current working directory")
	}
	hit := ""
	for _, c := range content {
		if strings.HasSuffix(c.Name(), internal.AppExtension) {
			if hit != "" {
				return "", fmt.Errorf("multiple applications found in current directory, specify the application name on the command line")
			}
			hit = c.Name()
		}
	}
	if hit == "" {
		return "", fmt.Errorf("no application found in current directory")
	}
	return filepath.Join(cwd, hit), nil
}

// extractImage extracts a docker application in a docker image to a temporary directory
func extractImage(appname string) (ExtractedApp, error) {
	var imagename string
	if strings.Contains(appname, ":") {
		nametag := strings.Split(appname, ":")
		if len(nametag) == 3 || strings.Contains(nametag[1], "/") {
			nametag[1] = internal.DirNameFromAppName(nametag[1])
			appname = filepath.Base(nametag[1])
		} else {
			nametag[0] = internal.DirNameFromAppName(nametag[0])
			appname = filepath.Base(nametag[0])
		}
		imagename = strings.Join(nametag, ":")
	} else {
		imagename = internal.DirNameFromAppName(appname)
		appname = filepath.Base(imagename)
	}
	tempDir, err := ioutil.TempDir("", "dockerapp")
	if err != nil {
		return ExtractedApp{}, errors.Wrap(err, "failed to create temporary directory")
	}
	defer os.RemoveAll(tempDir)
	err = Load(imagename, tempDir)
	if err != nil {
		if !strings.Contains(imagename, "/") {
			return ExtractedApp{}, fmt.Errorf("could not locate application in either filesystem or docker image")
		}
		// Try to pull it
		cmd := exec.Command("docker", "pull", imagename)
		if err := cmd.Run(); err != nil {
			return ExtractedApp{}, fmt.Errorf("could not locate application in filesystem, docker image or registry")
		}
		if err := Load(imagename, tempDir); err != nil {
			return ExtractedApp{}, errors.Wrap(err, "failed to load pulled image")
		}
	}
	// this gave us a compressed app, run through extract again
	app, err := Extract(filepath.Join(tempDir, appname))
	return ExtractedApp{
		OriginalAppName: "",
		AppName:         app.AppName,
		Cleanup:         app.Cleanup,
	}, err
}

// Extract extracts the app content if argument is an archive, or does nothing if a dir.
// It returns source file, effective app name, and cleanup function
// If appname is empty, it looks into cwd, and all subdirs for a single matching .dockerapp
// If nothing is found, it looks for an image and loads it
func Extract(appname string) (ExtractedApp, error) {
	if appname == "" {
		var err error
		if appname, err = findApp(); err != nil {
			return ExtractedApp{}, err
		}
	}
	if appname == "." {
		var err error
		if appname, err = os.Getwd(); err != nil {
			return ExtractedApp{}, errors.Wrap(err, "cannot resolve current working directory")
		}
	}
	originalAppname := appname
	appname = filepath.Clean(appname)
	// try appending our extension
	appname = internal.DirNameFromAppName(appname)
	s, err := os.Stat(appname)
	if err != nil {
		// try verbatim
		s, err = os.Stat(originalAppname)
	}
	if err != nil {
		// look for a docker image
		return extractImage(originalAppname)
	}
	if s.IsDir() {
		// directory: already decompressed
		return ExtractedApp{
			OriginalAppName: appname,
			AppName:         appname,
			Cleanup:         noop,
		}, nil
	}
	// not a dir: single-file or a tarball package, extract that in a temp dir
	tempDir, err := ioutil.TempDir("", "dockerapp")
	if err != nil {
		return ExtractedApp{}, errors.Wrap(err, "failed to create temporary directory")
	}
	defer func() {
		if err != nil {
			os.RemoveAll(tempDir)
		}
	}()
	appDir := filepath.Join(tempDir, filepath.Base(appname))
	if err = os.Mkdir(appDir, 0755); err != nil {
		return ExtractedApp{}, errors.Wrap(err, "failed to create application in temporary directory")
	}
	if err = extract(appname, appDir); err == nil {
		return ExtractedApp{
			OriginalAppName: appname,
			AppName:         appDir,
			Cleanup:         func() { os.RemoveAll(tempDir) },
		}, nil
	}
	if err = extractSingleFile(appname, appDir); err != nil {
		return ExtractedApp{}, err
	}
	// not a tarball, single-file then
	return ExtractedApp{
		OriginalAppName: appname,
		AppName:         appDir,
		Cleanup:         func() { os.RemoveAll(tempDir) },
	}, nil
}

func extractSingleFile(appname, appDir string) error {
	// not a tarball, single-file then
	data, err := ioutil.ReadFile(appname)
	if err != nil {
		return errors.Wrap(err, "failed to read single-file application package")
	}
	parts := strings.Split(string(data), "\n---")
	if len(parts) != 3 {
		return fmt.Errorf("malformed single-file application: expected 3 documents")
	}
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
		err = ioutil.WriteFile(filepath.Join(appDir, internal.FileNames[i]), []byte(data), 0644)
		if err != nil {
			return errors.Wrap(err, "failed to write application file")
		}
	}
	return nil
}

func extract(appname, outputDir string) error {
	f, err := os.Open(appname)
	if err != nil {
		return errors.Wrap(err, "failed to open application package")
	}
	defer f.Close()
	return archive.Untar(f, outputDir, &archive.TarOptions{
		NoLchown: true,
	})
}
