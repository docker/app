package loader

import (
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/app/internal"
	"github.com/docker/app/types"
	"github.com/docker/docker/pkg/archive"
	"github.com/pkg/errors"
)

// LoadFromURL loads a docker app from an URL that should return a single-file app.
func LoadFromURL(url string, ops ...func(*types.App) error) (*types.App, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, errors.Wrap(err, "failed to download "+url)
	}
	if resp.Body != nil {
		defer resp.Body.Close()
	}
	if resp.StatusCode != 200 {
		return nil, errors.Errorf("failed to download %s: %s", url, resp.Status)
	}
	return LoadFromSingleFile(url, resp.Body, ops...)
}

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
	// 2. is settings
	setting := strings.NewReader(parts[2])
	appOps := append([]func(*types.App) error{
		types.WithComposes(compose),
		types.WithSettings(setting),
		types.Metadata(metadata),
	}, ops...)
	return types.NewApp(path, appOps...)
}

// LoadFromDirectory loads a docker app from a directory
func LoadFromDirectory(path string, ops ...func(*types.App) error) (*types.App, error) {
	appOps := append([]func(*types.App) error{
		types.MetadataFile(filepath.Join(path, internal.MetadataFileName)),
		types.WithComposeFiles(filepath.Join(path, internal.ComposeFileName)),
		types.WithSettingsFiles(filepath.Join(path, internal.SettingsFileName)),
	}, ops...)
	return types.NewApp(path, appOps...)
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
