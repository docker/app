package packager

import (
	"fmt"
	"io/ioutil"
	"os"
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
				return "", fmt.Errorf("Error: multiple applications found in current directory, specify the application name on the command line")
			}
			hit = c.Name()
		}
	}
	if hit == "" {
		return "", fmt.Errorf("no application found in current directory")
	}
	return filepath.Join(cwd, hit), nil
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
		return nil, errors.Wrapf(err, "cannot locate application %q in filesystem", name)
	}
	if s.IsDir() {
		// directory: already decompressed
		appOpts := append(ops,
			types.WithPath(appname),
			types.WithSource(types.AppSourceSplit),
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
		ops = append(ops, types.WithSource(types.AppSourceMerged))
		return loader.LoadFromSingleFile(appname, f, ops...)
	}
	app.Source = types.AppSourceArchive
	return app, nil
}
