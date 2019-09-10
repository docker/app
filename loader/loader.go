package loader

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/docker/app/internal"
	"github.com/docker/app/types"
	"github.com/docker/docker/pkg/archive"
	"github.com/pkg/errors"
)

// LoadFromDirectory loads a docker app from a directory
func LoadFromDirectory(path string, ops ...func(*types.App) error) (*types.App, error) {
	if _, err := os.Stat(filepath.Join(path, internal.ParametersFileName)); os.IsNotExist(err) {
		if _, err := os.Stat(filepath.Join(path, internal.DeprecatedSettingsFileName)); err == nil {
			return nil, errors.Errorf("\"settings.yml\" has been deprecated in favor of \"parameters.yml\"; please rename \"settings.yml\" to \"parameters.yml\"")
		}
	}
	return types.NewAppFromDefaultFiles(path, ops...)
}

// LoadFromTar loads a docker app from a tarball
func LoadFromTar(tar string, ops ...func(*types.App) error) (*types.App, error) {
	f, err := os.Open(tar)
	if err != nil {
		return nil, errors.Wrap(err, "cannot load app from tar")
	}
	defer f.Close()
	appOps := append(ops, types.WithPath(tar))
	return LoadFromTarReader(f, appOps...)
}

// LoadFromTarReader loads a docker app from a tarball reader
func LoadFromTarReader(r io.Reader, ops ...func(*types.App) error) (*types.App, error) {
	dir, err := ioutil.TempDir("", "load-from-tar")
	if err != nil {
		return nil, errors.Wrap(err, "cannot load app from tar")
	}
	if err := archive.Untar(r, dir, &archive.TarOptions{
		NoLchown: true,
	}); err != nil {
		originalErr := errors.Wrap(err, "cannot load app from tar")
		if err := os.RemoveAll(dir); err != nil {
			return nil, errors.Wrapf(originalErr, "cannot remove temporary folder : %s", err.Error())
		}
		return nil, originalErr
	}
	appOps := append([]func(*types.App) error{
		types.WithCleanup(func() {
			os.RemoveAll(dir)
		}),
	}, ops...)
	return LoadFromDirectory(dir, appOps...)
}
