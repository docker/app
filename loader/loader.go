package loader

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/app/internal"
	"github.com/docker/app/types"
	"github.com/docker/docker/pkg/archive"
	"github.com/pkg/errors"
)

// LoadFromSingleFile loads a docker app from a single-file format (as a reader)
func LoadFromSingleFile(path string, r io.Reader, ops ...func(*types.App) error) (*types.App, error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, errors.Wrap(err, "error reading single-file")
	}
	parts := strings.Split(string(data), types.SingleFileSeparator)
	if len(parts) != 3 {
		return nil, errors.Errorf("malformed single-file application: expected 3 documents, got %d", len(parts))
	}
	// 0. is metadata
	metadata := strings.NewReader(parts[0])
	// 1. is compose
	compose := strings.NewReader(parts[1])
	// 2. is parameters
	parameters := strings.NewReader(parts[2])
	appOps := append([]func(*types.App) error{
		types.WithComposes(compose),
		types.WithParameters(parameters),
		types.Metadata(metadata),
	}, ops...)
	return types.NewApp(path, appOps...)
}

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
