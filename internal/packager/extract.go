package packager

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/docker/app/internal"
	"github.com/docker/app/loader"
	"github.com/docker/app/types"
	"github.com/pkg/errors"
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
func extractImage(appname string, ops ...func(*types.App) error) (*types.App, error) {
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
		return nil, errors.Wrap(err, "failed to create temporary directory")
	}
	defer os.RemoveAll(tempDir)
	err = Load(imagename, tempDir)
	if err != nil {
		if !strings.Contains(imagename, "/") {
			return nil, fmt.Errorf("could not locate application in either filesystem or docker image")
		}
		// Try to pull it
		cmd := exec.Command("docker", "pull", imagename)
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout
		if err := cmd.Run(); err != nil {
			return nil, fmt.Errorf("could not locate application in filesystem, docker image or registry")
		}
		if err := Load(imagename, tempDir); err != nil {
			return nil, errors.Wrap(err, "failed to load pulled image")
		}
	}
	return loader.LoadFromTar(filepath.Join(tempDir, appname), ops...)
}

// Extract extracts the app content if argument is an archive, or does nothing if a dir.
// It returns source file, effective app name, and cleanup function
// If appname is empty, it looks into cwd, and all subdirs for a single matching .dockerapp
// If nothing is found, it looks for an image and loads it
func Extract(name string, ops ...func(*types.App) error) (*types.App, error) {
	if name == "" {
		var err error
		if name, err = findApp(); err != nil {
			return nil, err
		}
	}
	if name == "." {
		var err error
		if name, err = os.Getwd(); err != nil {
			return nil, errors.Wrap(err, "cannot resolve current working directory")
		}
	}
	ops = append(ops, types.WithName(name))
	appname := internal.DirNameFromAppName(name)
	s, err := os.Stat(appname)
	if err != nil {
		// look for a docker image
		return extractImage(name, ops...)
	}
	if s.IsDir() {
		// directory: already decompressed
		appOpts := append(ops,
			types.WithPath(appname),
		)
		return loader.LoadFromDirectory(appname, appOpts...)
	}
	// not a dir: single-file or a tarball package, extract that in a temp dir
	app, err := loader.LoadFromTar(appname, ops...)
	if err != nil {
		f, err := os.Open(appname)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		return loader.LoadFromSingleFile(appname, f, ops...)
	}
	return app, nil
}
