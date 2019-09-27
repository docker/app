package packager

import (
	"github.com/docker/app/loader"
	"github.com/docker/app/types"
	"github.com/pkg/errors"
	"os"
)

// Extract extracts the app content if argument is an archive, or does nothing if a dir.
// It returns source file, effective app name, and cleanup function
func Extract(name string, ops ...func(*types.App) error) (*types.App, error) {
	if name == "." || name == "" {
		var err error
		if name, err = os.Getwd(); err != nil {
			return nil, errors.Wrap(err, "cannot resolve current working directory")
		}
	}
	ops = append(ops, types.WithName(name))
	s, err := os.Stat(name)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot locate application %q in filesystem", name)
	}
	if s.IsDir() {
		// directory: already decompressed
		appOpts := append(ops,
			types.WithPath(name),
			types.WithSource(types.AppSourceSplit),
		)
		return loader.LoadFromDirectory(name, appOpts...)
	}
	// not a dir: a tarball package, extract that in a temp dir
	app, err := loader.LoadFromTar(name, ops...)
	if err != nil {
		return nil, err
	}
	app.Source = types.AppSourceArchive
	return app, nil
}
